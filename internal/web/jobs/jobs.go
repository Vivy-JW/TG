// Package jobs 是「HTTP 触发的长任务」的唯一状态表。
//
// 为什么单独一层：TT 里同一件事（跑没跑、跑到哪、完了没）有三份互不兼容的实现 ——
// internal/dict/server 的 dbSyncJob、旧 internal/gp（已删）的 Job、internal/dict/server 的 mirrorJob，
// 前端要为三种形状写三种解析，三个轮询端点（/api/dbsync、/api/mirror、/api/gp/meta 的 job）。
// 而真正干活的功能包（internal/dict/dbsync、internal/mirror）**只接收一个进度回调**，
// 根本不需要知道自己"有个 job"。所以状态机归这里，功能包归功能包。
//
// 键（key）不限定成环境名：字典/镜像按环境跑，将来按别的维度跑也不会被卡住。
package jobs

import (
	"sync"
	"time"
)

// 状态取值。前端按这四个字符串分档，别改。
const (
	StateIdle    = "idle"
	StateRunning = "running"
	StateDone    = "done"
	StateError   = "error"
)

// Job 一个后台动作的进度。
//
// 字段的 JSON 名是与前端约定好的（设置页轮询 /api/gp/meta 的 job 字段），改动要连同前端一起改。
type Job struct {
	State     string         `json:"state"` // idle | running | done | error
	Kind      string         `json:"kind"`  // mirror | sync | …
	Done      int64          `json:"done,omitempty"`
	Total     int64          `json:"total,omitempty"`
	Progress  []string       `json:"progress,omitempty"`
	Error     string         `json:"error,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	Elapsed   string         `json:"elapsed,omitempty"`
	StartedAt string         `json:"startedAt,omitempty"`
	Started   time.Time      `json:"-"`
	Ended     time.Time      `json:"-"`
}

// maxProgress 进度行只留最近这么多条：一次同步会刷上百行，全留着没人看还占内存。
const maxProgress = 40

// Registry 一张 job 表。零值不可用，用 New。
type Registry struct {
	mu sync.Mutex
	m  map[string]*Job
}

// New 建一张空表。
func New() *Registry { return &Registry{m: map[string]*Job{}} }

// Get 取一份快照（带算好的 Elapsed/StartedAt）。没有这条记录时返回 idle。
func (r *Registry) Get(key string) *Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.m[key]
	if !ok {
		return &Job{State: StateIdle}
	}
	cp := *j
	cp.Progress = append([]string(nil), j.Progress...)
	if !j.Started.IsZero() {
		end := j.Ended
		if end.IsZero() {
			end = time.Now()
		}
		cp.Elapsed = end.Sub(j.Started).Round(time.Second).String()
		cp.StartedAt = j.Started.Format(time.RFC3339)
	}
	return &cp
}

// Update 原地改一条记录（没有就建）。所有写操作都走这里，锁只加一处。
func (r *Registry) Update(key string, f func(*Job)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.m[key]
	if !ok {
		j = &Job{State: StateIdle}
		r.m[key] = j
	}
	f(j)
}

// Running 这个 key 上是否有任务正在跑（同 key 同时只允许一个）。
func (r *Registry) Running(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	j, ok := r.m[key]
	return ok && j.State == StateRunning
}

// Start 把 key 置为 running，并清掉上一轮的结果与进度。
func (r *Registry) Start(key, kind string) {
	r.Update(key, func(j *Job) {
		*j = Job{State: StateRunning, Kind: kind, Started: time.Now()}
	})
}

// Note 追加一行进度文字。
func (r *Registry) Note(key, msg string) {
	if msg == "" {
		return
	}
	r.Update(key, func(j *Job) {
		j.Progress = append(j.Progress, msg)
		if len(j.Progress) > maxProgress {
			j.Progress = j.Progress[len(j.Progress)-maxProgress:]
		}
	})
}

// Count 更新「已完成 / 总数」，message 非空时同时记一行进度（镜像与字典同步共用）。
func (r *Registry) Count(key string, done, total int, msg string) {
	r.Update(key, func(j *Job) {
		j.Done = int64(done)
		j.Total = int64(total)
		if msg == "" {
			return
		}
		j.Progress = append(j.Progress, msg)
		if len(j.Progress) > maxProgress {
			j.Progress = j.Progress[len(j.Progress)-maxProgress:]
		}
	})
}

// Finish 收尾：err 非空记 error，否则记 done + result，并打上结束时间。
func (r *Registry) Finish(key string, result map[string]any, err error) {
	r.Update(key, func(j *Job) {
		j.Ended = time.Now()
		j.Result = result
		if err != nil {
			j.State = StateError
			j.Error = err.Error()
			return
		}
		j.State = StateDone
		j.Error = ""
	})
}
