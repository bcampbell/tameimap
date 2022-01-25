# tameimap

A simple IMAP server which serves up email from local files, for testing.

## Installation

Requires golang to compile and install:

```
$ go install github.com/bcampbell/tameimap
```

I can probably put up a binary release if anyone really needs it - get in contact.

## Usage

```
Usage: ./tameimap [OPTIONS]
A simple IMAP server.
Options:
  -d string
    	dump all network traffic to file ("-" = stdout)
  -p int
    	port number (default 1143)
  -v	output extra info to stderr
```


There's currently a hardcoded user 'bob' built in (password: "pass"), with `./bob` as it's root directory.

## Example

Set up a test directory, eg:

```
bob
└── INBOX
    └── hello.eml
```

Where hello.eml is an email in RFC822 format, something like:

```
Date: Tue, 25 Jan 2020 23:11:08 +0000
Message-ID: <1642365537868884386>
Subject: Welcome to tameimap
From: bob@localhost
To: bob@localhost

Here's a placeholder message just to show there's something here.

```


Run the server (dumping net traffic to stdout):
```
$ tameimap -d -
```

Then connect an IMAP client.

As a quick test, using netcat from another terminal:
```
$ echo -n -e "a LOGIN bob pass\r\nb SELECT INBOX\r\nc LOGOUT\r\n" | nc localhost 1143
```

You should see something like:
```
* OK [CAPABILITY IMAP4rev1 LITERAL+ SASL-IR CHILDREN UNSELECT MOVE APPENDLIMIT AUTH=PLAIN] IMAP4rev1 Service Ready
a LOGIN bob pass
b SELECT INBOX
c LOGOUT
a OK [CAPABILITY IMAP4rev1 LITERAL+ SASL-IR CHILDREN UNSELECT MOVE APPENDLIMIT] LOGIN completed
* OK [PERMANENTFLAGS (\Answered \Flagged \Deleted \Seen \Draft \*)] Flags permitted.
* OK [UNSEEN 1] Message 1 is first unseen
* 0 RECENT
* OK [UIDNEXT 2] Predicted next UID
* OK [UIDVALIDITY 1] UIDs valid
* 1 EXISTS
b OK [READ-WRITE] SELECT completed
* BYE Closing connection
c OK LOGOUT completed
```


