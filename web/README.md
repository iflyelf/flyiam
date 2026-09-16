# FlyIAM Web Frontend

FlyIAM 统一用户管理系统的前端项目，基于 Vue 3 + Vite + Element Plus 构建。

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Vite 5** - 下一代前端构建工具
- **Element Plus** - Vue 3 UI 组件库
- **Vue Router 4** - 官方路由管理器
- **Pinia** - Vue 3 状态管理
- **Axios** - HTTP 客户端
- **Sass** - CSS 预处理器

## 功能特性

- ✅ **仪表盘** - 系统概览、统计数据、快速操作
- ✅ **用户管理** - 用户列表、搜索、详情查看
- ✅ **数据同步** - 数据源同步、Casdoor 同步、完整同步
- ✅ **同步日志** - 日志列表、筛选、详情查看
- ✅ **系统设置** - 系统配置、数据库配置、Redis 配置
- ✅ **响应式布局** - 支持桌面端和移动端
- ✅ **暗色模式** - 深色主题支持
- ✅ **国际化** - 中文界面

## 快速开始

### 安装依赖

```bash
npm install
```

### 开发模式

```bash
npm run dev
```

访问 http://localhost:3000

### 生产构建

```bash
npm run build
```

构建产物输出到 `dist/` 目录。

### 预览生产构建

```bash
npm run preview
```

## 项目结构

```
web/
├── public/              # 静态资源
├── src/
│   ├── assets/         # 资源文件
│   │   └── styles/     # 样式文件
│   ├── components/     # 公共组件
│   │   └── Layout.vue  # 布局组件
│   ├── router/         # 路由配置
│   ├── stores/         # Pinia 状态管理
│   ├── api/            # API 请求
│   ├── utils/          # 工具函数
│   ├── views/          # 页面组件
│   │   ├── Dashboard.vue   # 仪表盘
│   │   ├── Users.vue       # 用户管理
│   │   ├── Sync.vue        # 数据同步
│   │   ├── Logs.vue        # 同步日志
│   │   └── Settings.vue    # 系统设置
│   ├── App.vue         # 根组件
│   └── main.js         # 入口文件
├── index.html          # HTML 模板
├── vite.config.js      # Vite 配置
└── package.json        # 项目配置
```

## 环境变量

创建 `.env.local` 文件配置环境变量：

```bash
# API 地址
VITE_API_BASE_URL=http://localhost:8081

# 是否启用 Mock 数据
VITE_USE_MOCK=false
```

## 代理配置

开发环境 API 代理配置在 `vite.config.js`：

```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8081',
      changeOrigin: true
    }
  }
}
```

## 页面路由

| 路径 | 页面 | 说明 |
|------|------|------|
| `/` | Dashboard | 仪表盘（重定向） |
| `/dashboard` | Dashboard | 系统概览 |
| `/users` | Users | 用户管理 |
| `/sync` | Sync | 数据同步 |
| `/logs` | Logs | 同步日志 |
| `/settings` | Settings | 系统设置 |

## 组件说明

### Layout.vue

主布局组件，包含：
- 侧边栏导航
- 顶部栏（面包屑、用户信息、主题切换）
- 内容区域
- 支持侧边栏折叠
- 响应式适配

### Dashboard.vue

仪表盘页面，展示：
- 统计卡片（在职用户、离职用户、未同步用户、今日同步）
- 最近同步记录
- 系统状态
- 快速操作

### Users.vue

用户管理页面，功能：
- 用户列表展示
- 搜索功能
- 分页
- 用户详情查看
- 单个用户同步

### Sync.vue

数据同步页面，功能：
- 数据源同步
- Casdoor 同步
- 完整同步
- 同步进度显示
- 配置管理

### Logs.vue

同步日志页面，功能：
- 日志列表
- 筛选（类型、状态、日期）
- 分页
- 详情查看
- 失败重试

### Settings.vue

系统设置页面，包含：
- 系统配置
- 数据库配置
- Redis 配置
- 审计配置
- 关于信息

## 构建优化

- 代码分割（Code Splitting）
- 按需加载（Lazy Loading）
- Chunk 优化
- 资源压缩

## 浏览器支持

- Chrome >= 87
- Firefox >= 78
- Safari >= 14
- Edge >= 88

## 开发规范

- 使用 Composition API
- 组件采用 `<script setup>` 语法
- 样式使用 Scoped SCSS
- 遵循 Vue 3 最佳实践

## 相关文档

- [Vue 3 文档](https://vuejs.org/)
- [Vite 文档](https://vitejs.dev/)
- [Element Plus 文档](https://element-plus.org/)
- [Vue Router 文档](https://router.vuejs.org/)

## 许可证

MIT License
