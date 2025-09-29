imapdb
------

A key-value pair storage via IMAP mailboxes.

## Installation

```bash
go install github.com/lkebin/imapdb/cmd/imapdb@main
```


## Usage

```bash
export IMAP_SERVER=${your_imap_server:port}
export IMAP_USER=${your_email}
export IMAP_PASSWORD=${your_password}

imapdb set key value
imapdb get key
imapdb delete key
```
