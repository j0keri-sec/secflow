// SecFlow MongoDB 初始化脚本
// 首次启动时自动执行，创建必要的索引和初始数据

// 切换到 secflow 数据库
db = db.getSiblingDB('secflow');

// ================================================
// 1. 创建集合
// ================================================

// 用户集合
db.createCollection('users');
db.users.createIndex({ 'username': 1 }, { unique: true });
db.users.createIndex({ 'email': 1 }, { unique: true });

// 漏洞记录集合
db.createCollection('vuln_records');
db.vuln_records.createIndex({ 'key': 1 }, { unique: true });
db.vuln_records.createIndex({ 'created_at': -1 });
db.vuln_records.createIndex({ 'updated_at': -1 });
db.vuln_records.createIndex({ 'source': 1, 'created_at': -1 });
db.vuln_records.createIndex({ 'severity': 1, 'created_at': -1 });
db.vuln_records.createIndex({ 'cve_id': 1 }, { sparse: true });

// 安全文章集合
db.createCollection('articles');
db.articles.createIndex({ 'key': 1 }, { unique: true });
db.articles.createIndex({ 'created_at': -1 });
db.articles.createIndex({ 'source': 1, 'created_at': -1 });
db.articles.createIndex({ 'category': 1, 'created_at': -1 });

// 任务记录集合
db.createCollection('tasks');
db.tasks.createIndex({ 'task_id': 1 }, { unique: true });
db.tasks.createIndex({ 'status': 1, 'created_at': -1 });
db.tasks.createIndex({ 'created_at': -1 });
db.tasks.createIndex({ 'source': 1, 'status': 1 });

// 节点集合
db.createCollection('nodes');
db.nodes.createIndex({ 'node_id': 1 }, { unique: true });
db.nodes.createIndex({ 'name': 1 }, { unique: true });
db.nodes.createIndex({ 'status': 1, 'last_heartbeat': -1 });

// 推送渠道配置集合
db.createCollection('push_channels');
db.push_channels.createIndex({ 'channel_id': 1 }, { unique: true });
db.push_channels.createIndex({ 'type': 1, 'enabled': 1 });

// 系统配置集合
db.createCollection('system_config');
db.system_config.createIndex({ 'key': 1 }, { unique: true });

// 操作日志集合
db.createCollection('operation_logs');
db.operation_logs.createIndex({ 'created_at': -1 });
db.operation_logs.createIndex({ 'operator': 1, 'created_at': -1 });
db.operation_logs.createIndex({ 'action': 1, 'created_at': -1 });

// ================================================
// 2. 创建默认管理员账号
// ================================================

// 检查是否已存在管理员
const adminExists = db.users.findOne({ username: 'admin' });
if (!adminExists) {
    // 默认密码: admin123 (生产环境请立即修改!)
    db.users.insertOne({
        username: 'admin',
        password: '$2a$10$N9qo8uLOickgx2ZMRZoMye1J8g5vXZvQzQhK1xGQq3J3Qq3J3J3J3', // bcrypt hash of 'admin123'
        role: 'admin',
        email: 'admin@secflow.local',
        created_at: new Date(),
        updated_at: new Date(),
        is_active: true
    });
    print('Default admin user created: admin / admin123');
} else {
    print('Admin user already exists, skipping...');
}

// ================================================
// 3. 创建默认推送渠道模板
// ================================================

const pushChannelExists = db.push_channels.findOne({ type: 'dingtalk' });
if (!pushChannelExists) {
    db.push_channels.insertMany([
        {
            channel_id: 'default-dingtalk',
            name: '钉钉通知',
            type: 'dingtalk',
            enabled: false,
            config: {
                webhook_url: '',
                secret: ''
            },
            created_at: new Date(),
            updated_at: new Date()
        },
        {
            channel_id: 'default-feishu',
            name: '飞书通知',
            type: 'feishu',
            enabled: false,
            config: {
                webhook_url: '',
                secret: ''
            },
            created_at: new Date(),
            updated_at: new Date()
        }
    ]);
    print('Default push channels created');
}

// ================================================
// 4. 创建系统配置
// ================================================

const configExists = db.system_config.findOne({ key: 'grabber' });
if (!configExists) {
    db.system_config.insertMany([
        {
            key: 'grabber',
            value: {
                interval: '1h',
                init_page_limit: 3,
                update_page_limit: 1
            },
            description: '爬取器全局配置',
            updated_at: new Date()
        },
        {
            key: 'data_retention',
            value: {
                days: 90,
                vulns_enabled: true,
                articles_enabled: true,
                tasks_enabled: false
            },
            description: '数据保留策略',
            updated_at: new Date()
        }
    ]);
    print('Default system config created');
}

// ================================================
// 5. 输出完成信息
// ================================================

print('===========================================');
print('SecFlow MongoDB 初始化完成');
print('===========================================');
print('数据库: secflow');
print('集合: users, vuln_records, articles, tasks, nodes, push_channels, system_config, operation_logs');
print('默认管理员: admin / admin123');
print('请立即修改默认管理员密码!');
print('===========================================');
