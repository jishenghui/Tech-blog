# deploy

部署模板，尚未绑定真实域名与证书。

- `nginx.conf`：ECS 上的 `conf.d` 片段，只服务 `api.` 与 `admin.` 子域；主站 `www.` 由七牛 CDN + Kodo 承载。CI 会复制到 `/etc/nginx/conf.d/blog.conf`。
- `blog-api.service`：Go 服务的 systemd unit，安装到 `/etc/systemd/system/blog-api.service`。
- `env/*.env.example`：后端、前台、后台的环境变量模板；复制到对应项目后填写，不要提交真实密钥。

上线前替换所有 `yourdomain.com`，并在 ECS 上准备 `/opt/blog/`、`/opt/blog/admin/`、`/opt/blog/data/` 与 `/opt/blog/.env`。
