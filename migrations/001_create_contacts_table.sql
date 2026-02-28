CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phone_number TEXT,
    email TEXT,
    linked_id INTEGER,
    link_precedence TEXT CHECK(link_precedence IN ('primary', 'secondary')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY (linked_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_phone ON contacts(phone_number);
CREATE INDEX IF NOT EXISTS idx_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_linked_id ON contacts(linked_id);
