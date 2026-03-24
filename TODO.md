# SecFlow 开发计划

## 📋 当前状态

### ✅ 已完成
- [x] 密码重置系统
- [x] 用户权限系统完善
- [x] 单元测试 (auth_test.go, handler_test.go)
- [x] CHANGELOG.md
- [x] 监控告警系统 (AlertManager 配置)
- [x] 运维脚本 (backup.sh, restore.sh, healthcheck.sh)

### 🔄 进行中
- [ ] Docker 优化

### 📌 待完成

#### 1. 测试完善
- [x] 为 secflow-server 编写更多单元测试 ✅
- [x] handler 测试 ✅
- [ ] 为 secflow-client 编写基本测试
- [ ] 添加集成测试脚本

#### 2. 监控告警系统
- [x] AlertManager 配置 ✅
- [x] 告警规则完善 ✅ (已有 alerts.yml)
- [ ] 优化 Grafana dashboard
- [ ] 添加日志收集配置 (ELK/Loki)

#### 3. 运维功能
- [x] 备份脚本 (backup.sh) ✅
- [x] 恢复脚本 (restore.sh) ✅
- [x] 健康检查脚本 (healthcheck.sh) ✅
- [ ] 日志收集配置
- [ ] 数据迁移脚本

#### 4. Docker 优化
- [ ] 添加 .dockerignore
- [ ] 优化镜像大小
- [ ] 添加 Helm Chart (K8s)

#### 5. 代码质量
- [ ] 为所有公开函数添加 godoc 注释
- [ ] 检查并修复代码中的 TODO
- [ ] 添加更多示例代码

#### 6. 文档完善
- [ ] 补充快速入门文档
- [ ] 添加 API 使用示例
- [ ] 添加故障排查指南

---

## 🚀 执行计划

### Phase 1: 测试完善 (当前)
1. 编写 handler 单元测试
2. 编写 repository 单元测试
3. 添加集成测试脚本

### Phase 2: 监控告警
1. 完善 Prometheus 配置
2. 添加 AlertManager
3. 优化 Grafana

### Phase 3: 运维脚本
1. 备份脚本
2. 迁移脚本
3. 健康检查

### Phase 4: Docker & 部署
1. 优化 Dockerfile
2. 添加 Helm Chart
3. 完善 docker-compose

### Phase 5: 代码质量 & 文档
1. 添加 godoc 注释
2. 修复 TODO
3. 完善文档
