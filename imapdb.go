package imapdb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"
)

var ErrNotFound = errors.New("not found")

type Option func(*options)
type options struct {
	dbName string
}

func WithDBName(name string) Option {
	return func(o *options) {
		o.dbName = name
	}
}

type DB interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Close() error
}

type dbImpl struct {
	client *imapclient.Client
	dbName string
}

func NewDB(address, user, pass string, opts ...Option) (DB, error) {
	o := &options{
		dbName: "imapdb",
	}

	for _, opt := range opts {
		opt(o)
	}

	c, err := imapclient.DialTLS(address, nil)
	if err != nil {
		return nil, err
	}

	if err := c.Login(user, pass).Wait(); err != nil {
		return nil, err
	}

	if _, err := c.Select(o.dbName, nil).Wait(); err != nil {
		// try to create mailbox
		if err := c.Create(o.dbName, nil).Wait(); err != nil {
			c.Close()
			return nil, fmt.Errorf("failed to create mailbox %s: %v", o.dbName, err)
		}
	}

	return &dbImpl{client: c, dbName: o.dbName}, nil
}

func (db *dbImpl) Get(key string) ([]byte, error) {
	if _, err := db.client.Select(db.dbName, nil).Wait(); err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %v", err)
	}

	search, err := db.client.UIDSearch(&imap.SearchCriteria{Text: []string{key}}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to search messages in %s: %v", db.dbName, err)
	}

	if len(search.AllUIDs()) == 0 {
		return nil, ErrNotFound
	}

	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}
	fetchCmd := db.client.Fetch(imap.UIDSetNum(search.AllUIDs()...), fetchOptions)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, errors.New("FETCH command did not return any message")
	}

	var bodySectionData imapclient.FetchItemDataBodySection
	ok := false
	for {
		item := msg.Next()
		if item == nil {
			break
		}
		bodySectionData, ok = item.(imapclient.FetchItemDataBodySection)
		if ok {
			break
		}
	}
	if !ok {
		return nil, errors.New("FETCH command did not return body section")
	}

	mr, err := mail.CreateReader(bodySectionData.Literal)
	if err != nil {
		return nil, fmt.Errorf("failed to create mail reader: %v", err)
	}

	var data []byte

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read message part: %v", err)
		}

		switch p.Header.(type) {
		case *mail.AttachmentHeader:
			data, err = io.ReadAll(p.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read attachment body: %v", err)
			}
		}
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to close fetch command: %v", err)
	}

	if data == nil {
		return nil, ErrNotFound
	}
	return data, nil
}

func (db *dbImpl) Set(key string, value []byte) error {
	if _, err := db.client.Select(db.dbName, nil).Wait(); err != nil {
		return err
	}

	search, err := db.client.UIDSearch(
		&imap.SearchCriteria{Text: []string{key}},
		nil,
	).Wait()
	if err != nil {
		return fmt.Errorf("failed to search messages in %s: %v", db.dbName, err)
	}

	if len(search.AllUIDs()) > 0 {
		// delete first
		db.client.Store(imap.UIDSetNum(search.AllUIDs()...), &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}, nil)
		db.client.Expunge().Close()
	}

	var b bytes.Buffer
	var h mail.Header
	h.SetDate(time.Now())
	h.SetSubject(key)

	m, err := mail.CreateWriter(&b, h)
	if err != nil {
		return fmt.Errorf("failed to create mail writer: %v", err)
	}

	// body
	var th mail.InlineHeader
	th.Set("Content-Type", "text/plain; charset=utf-8")

	tw, err := m.CreateSingleInline(th)
	if err != nil {
		return fmt.Errorf("failed to create inline part: %v", err)
	}
	if _, err := tw.Write([]byte(key)); err != nil {
		return fmt.Errorf("failed to write inline part: %v", err)
	}
	tw.Close()

	// attachment
	var a mail.AttachmentHeader
	a.Set("Content-Type", "application/octet-stream")
	a.SetFilename("data.bin")
	aw, err := m.CreateAttachment(a)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %v", err)
	}
	if _, err := aw.Write(value); err != nil {
		return fmt.Errorf("failed to write attachment: %v", err)
	}
	aw.Close()
	m.Close()

	appendCmd := db.client.Append(db.dbName, int64(b.Len()), nil)
	if _, err := appendCmd.Write(b.Bytes()); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}
	if err := appendCmd.Close(); err != nil {
		return fmt.Errorf("failed to close message: %v", err)
	}
	if _, err := appendCmd.Wait(); err != nil {
		return fmt.Errorf("APPEND command failed: %v", err)
	}
	return nil
}

func (db *dbImpl) Delete(key string) error {
	if _, err := db.client.Select(db.dbName, nil).Wait(); err != nil {
		return err
	}

	search, err := db.client.UIDSearch(&imap.SearchCriteria{Text: []string{key}}, nil).Wait()
	if err != nil {
		return fmt.Errorf("failed to search messages in %s: %v", db.dbName, err)
	}

	if len(search.AllUIDs()) > 0 {
		db.client.Store(imap.UIDSetNum(search.AllUIDs()...), &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}, nil)
		db.client.Expunge().Close()
	}

	return nil
}

func (db *dbImpl) Close() error {
	return db.client.Close()
}
