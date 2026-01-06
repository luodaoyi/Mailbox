# MailBox SMTP 测试

## 快速开始

### 1. 创建虚拟环境

```bash
uv venv
```

### 2. 激活虚拟环境

Windows:
```bash
.venv\Scripts\activate
```

Linux/Mac:
```bash
source .venv/bin/activate
```

### 3. 运行测试脚本

```bash
python test_smtp.py
```

## 前置条件

- 确保本地 SMTP 服务运行在 `localhost:25`
- 在管理后台启用 `cc.com` 域名
- Python >= 3.10

## 测试内容

脚本会发送一封测试邮件到 `test@cc.com`，包含：
- 纯文本内容
- HTML 内容
- 文本附件 (test.txt)
