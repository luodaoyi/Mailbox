-- 修复外键约束，添加级联删除
USE mailbox;

-- 删除旧的外键约束
ALTER TABLE attachments DROP FOREIGN KEY fk_emails_attachments;

-- 添加新的外键约束，带级联删除
ALTER TABLE attachments
ADD CONSTRAINT fk_emails_attachments
FOREIGN KEY (email_id) REFERENCES emails(id)
ON DELETE CASCADE;
