#!/usr/bin/env python3
"""SMTP 压测脚本 - 测试邮件系统负载和接收速度"""

import smtplib
import random
import time
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.mime.base import MIMEBase
from email import encoders

# 压测配置
TOTAL_EMAILS = 1000      # 总邮件数
CONCURRENT = 10          # 并发数
MAILBOX_RANGE = 100      # 邮箱范围 (test1-test100)

# 统计数据
stats = {
    'success': 0,
    'failed': 0,
    'total_time': 0,
    'lock': threading.Lock()
}


def send_test_email(index):
    """发送测试邮件到随机邮箱"""
    start_time = time.time()

    # 随机选择邮箱
    mailbox_num = random.randint(1, MAILBOX_RANGE)
    recipient = f"test{mailbox_num}@cc.com"

    sender = "sender@example.com"
    subject = f"压测邮件 #{index}"

    msg = MIMEMultipart("alternative")
    msg["From"] = sender
    msg["To"] = recipient
    msg["Subject"] = subject

    text_content = f"压测邮件 #{index}\n发送到: {recipient}"
    msg.attach(MIMEText(text_content, "plain", "utf-8"))

    html_content = f"""
    <html>
      <body>
        <h2>压测邮件 #{index}</h2>
        <p>收件人: {recipient}</p>
      </body>
    </html>
    """
    msg.attach(MIMEText(html_content, "html", "utf-8"))

    try:
        with smtplib.SMTP("localhost", 25, timeout=10) as server:
            server.send_message(msg)

        elapsed = time.time() - start_time
        with stats['lock']:
            stats['success'] += 1
            stats['total_time'] += elapsed

        return True, index, recipient, elapsed
    except Exception as e:
        elapsed = time.time() - start_time
        with stats['lock']:
            stats['failed'] += 1

        return False, index, recipient, str(e)


if __name__ == "__main__":
    print(f"=== SMTP 压测开始 ===")
    print(f"总邮件数: {TOTAL_EMAILS}")
    print(f"并发数: {CONCURRENT}")
    print(f"邮箱范围: test1@cc.com - test{MAILBOX_RANGE}@cc.com")
    print(f"{'='*50}\n")

    start_time = time.time()

    with ThreadPoolExecutor(max_workers=CONCURRENT) as executor:
        futures = [executor.submit(send_test_email, i) for i in range(1, TOTAL_EMAILS + 1)]

        completed = 0
        for future in as_completed(futures):
            completed += 1
            result = future.result()

            if result[0]:
                print(f"[{completed}/{TOTAL_EMAILS}] ✓ 邮件 #{result[1]} -> {result[2]} ({result[3]:.3f}s)")
            else:
                print(f"[{completed}/{TOTAL_EMAILS}] ✗ 邮件 #{result[1]} 失败: {result[3]}")

            if completed % 100 == 0:
                elapsed = time.time() - start_time
                rate = completed / elapsed
                print(f"\n--- 进度: {completed}/{TOTAL_EMAILS} | 速度: {rate:.2f} 封/秒 ---\n")

    total_elapsed = time.time() - start_time

    print(f"\n{'='*50}")
    print(f"=== 压测完成 ===")
    print(f"总耗时: {total_elapsed:.2f} 秒")
    print(f"成功: {stats['success']} 封")
    print(f"失败: {stats['failed']} 封")
    print(f"成功率: {stats['success']/TOTAL_EMAILS*100:.2f}%")
    print(f"平均速度: {TOTAL_EMAILS/total_elapsed:.2f} 封/秒")
    if stats['success'] > 0:
        print(f"平均延迟: {stats['total_time']/stats['success']*1000:.2f} ms")
    print(f"{'='*50}")
