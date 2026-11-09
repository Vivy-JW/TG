package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// GP 环境与 T100 共用 hosts.sshs；T100 的字典同步/镜像接口必须按 product 拒绝 GP，
// 否则直接调 API 会把 GP 当成 T100 环境写库/拉镜像。
func TestT100EndpointsRejectGP(t *testing.T) {
	p := writeTempConfig(t, `{"hosts":{"sshs":[
      {"name":"t100","host":"1.2.3.4","user":"u","password":"p",
       "db":{"type":"oracle","host":"1.2.3.4","port":1521,"service":"s","accounts":[{"account":"x","password":"x"}]}},
      {"name":"gp","host":"5.6.7.8","user":"u","password":"p","product":"gp",
       "db":{"type":"oracle","host":"5.6.7.8","port":1521,"service":"g","accounts":[{"account":"y","password":"y"}]}}]}}`)
	s := New(p)

	rec := httptest.NewRecorder()
	s.hDBSyncGet(rec, httptest.NewRequest("GET", "/api/dbsync", nil))
	var syncResp dbSyncResp
	if err := json.Unmarshal(rec.Body.Bytes(), &syncResp); err != nil {
		t.Fatal(err)
	}
	if len(syncResp.Envs) != 1 || syncResp.Envs[0].Name != "t100" {
		t.Fatalf("T100 同步清单应只含 t100，得到 %+v", syncResp.Envs)
	}

	rec = httptest.NewRecorder()
	s.hDBSyncPost(rec, httptest.NewRequest("POST", "/api/dbsync", bytes.NewBufferString(`{"env":"gp"}`)))
	if rec.Code != 400 {
		t.Fatalf("GP 走 T100 dbsync 应 400，得到 %d: %s", rec.Code, rec.Body.String())
	}
	if s.sync.Running {
		t.Fatal("被拒绝的 GP 同步不应启动任务")
	}

	rec = httptest.NewRecorder()
	s.hMirrorGet(rec, httptest.NewRequest("GET", "/api/mirror", nil))
	var mirrorResp mirrorResp
	if err := json.Unmarshal(rec.Body.Bytes(), &mirrorResp); err != nil {
		t.Fatal(err)
	}
	if len(mirrorResp.Envs) != 1 || mirrorResp.Envs[0].Name != "t100" {
		t.Fatalf("T100 镜像清单应只含 t100，得到 %+v", mirrorResp.Envs)
	}

	rec = httptest.NewRecorder()
	s.hMirrorPull(rec, httptest.NewRequest("POST", "/api/mirror/pull", bytes.NewBufferString(`{"env":"gp"}`)))
	if rec.Code != 400 {
		t.Fatalf("GP 走 T100 mirror 应 400，得到 %d: %s", rec.Code, rec.Body.String())
	}
	if s.mirror.Running {
		t.Fatal("被拒绝的 GP 镜像不应启动任务")
	}
}
