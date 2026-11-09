---
name: tg-pj-git
description: 在 TG 中涉及任何代码修改和 git 相关的命令时，请激活此技能来加载项目管理规范。触发词：分支、commit、PR、推送、提交规范。
---

# TG 的项目规范

涉及任何修改必须创建自己的分支，完成工作后向远程仓库推送PR

## commit

### commit内容要求

标题必须按照下面的格式

```
feat(scope): title
fix(scope): title
chore(scope): title
refactor(scope): title
```

> Body、Footer内容格式不设要求
