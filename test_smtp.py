#!/usr/bin/env python3
"""SMTP 测试脚本 - 发送邮件到本地服务器"""

import smtplib
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.mime.base import MIMEBase
from email import encoders
from io import BytesIO


def send_test_email():
    """发送测试邮件到 localhost:25"""

    # 邮件配置
    sender = "sender@example.com"
    recipient = "test@cc.com"
    subject = "测试邮件 - SMTP Test"

    # 创建邮件
    msg = MIMEMultipart("alternative")
    msg["From"] = sender
    msg["To"] = recipient
    msg["Subject"] = subject

    # 纯文本内容
    text_content = "这是纯文本内容\n\nSMTP 测试邮件"
    msg.attach(MIMEText(text_content, "plain", "utf-8"))

    # HTML 内容
    html_content = """
    <html>
      <body>
        <h2>这是 HTML 内容</h2>
        <p>SMTP 测试邮件</p>
        <p><strong>测试成功！</strong></p>
      </body>
    </html>
    """
    msg.attach(MIMEText(html_content, "html", "utf-8"))

    # 添加文本附件
    attachment_content = "这是附件内容\n测试附件文件"
    part = MIMEBase("application", "octet-stream")
    part.set_payload(attachment_content.encode("utf-8"))
    encoders.encode_base64(part)
    part.add_header("Content-Disposition", "attachment; filename=test.txt")
    msg.attach(part)

    # 发送邮件
    try:
        with smtplib.SMTP("localhost", 25) as server:
            server.send_message(msg)
            print(f"✓ 邮件发送成功: {sender} -> {recipient}")
    except Exception as e:
        print(f"✗ 发送失败: {e}")
        raise


if __name__ == "__main__":
    send_test_email()
