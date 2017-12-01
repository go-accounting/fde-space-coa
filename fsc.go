package fdespacecoa

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"time"

	"github.com/go-accounting/coa"
	"github.com/go-accounting/deb"
	"github.com/go-accounting/fde"
)

type integrations struct {
	space deb.Space
	cr    *coa.CoaRepository
	coaid *string
}

type transactionMetadata struct {
	Memo    string
	Tags    []string
	User    string
	Removes int64
}

func NewStoreAndAccountsRepository(space deb.Space, cr *coa.CoaRepository, coaid *string) (fde.Store, fde.AccountsRepository, error) {
	integrations := &integrations{space, cr, coaid}
	return integrations, integrations, nil
}

func (s *integrations) Get(txid string) (*fde.Transaction, error) {
	id, err := strconv.Atoi(txid)
	if err != nil {
		return nil, err
	}
	ts, err := s.space.Slice(nil, nil, []deb.MomentRange{deb.MomentRange{deb.Moment(id), deb.Moment(id)}})
	if err != nil {
		return nil, err
	}
	ch, errch := ts.Transactions()
	var result *fde.Transaction
	for t := range ch {
		result, err = s.debTransactionToFdeTransaction(t)
	}
	if err == nil {
		err = <-errch
	} else {
		<-errch
	}
	return result, err
}

func (s *integrations) Append(tt ...*fde.Transaction) ([]string, error) {
	result := make([]string, len(tt))
	now := time.Now()
	ch := make(chan *deb.Transaction)
	errch := make(chan error, 1)
	go func() {
		var err error
		for i, t := range tt {
			var dt *deb.Transaction
			dt, err = s.fdeTransactionToDebTransaction(t, deb.MomentFromTime(now)+deb.Moment(i))
			if err != nil {
				break
			}
			ch <- dt
			result[i] = fmt.Sprint(dt.Moment)
		}
		close(ch)
		errch <- err
	}()
	err := s.space.Append(deb.ChannelSpace(ch))
	if err == nil {
		return result, <-errch
	} else {
		return nil, err
	}
}

func (ar *integrations) Exists(ids []string) ([]bool, error) {
	result := make([]bool, len(ids))
	idxs, err := ar.cr.Indexes(*ar.coaid, ids, []string{"detail"})
	if err != nil {
		return nil, err
	}
	for i, idx := range idxs {
		result[i] = idx != -1
	}
	return result, nil
}

func (i *integrations) debTransactionToFdeTransaction(t *deb.Transaction) (*fde.Transaction, error) {
	aa, err := i.cr.AllAccounts(*i.coaid)
	if err != nil {
		return nil, err
	}
	d := t.Date.ToTime()
	m := t.Moment.ToTime()
	buf := bytes.NewBuffer(t.Metadata)
	dec := gob.NewDecoder(buf)
	var tm transactionMetadata
	if err := dec.Decode(&tm); err != nil {
		return nil, err
	}
	deb := fde.Entries{}
	cre := fde.Entries{}
	for k, v := range t.Entries {
		if v > 0 {
			deb = append(deb, fde.Entry{Account: aa[k-1].Id, Value: v})
		} else {
			cre = append(cre, fde.Entry{Account: aa[k-1].Id, Value: -v})
		}
	}
	removes := ""
	if tm.Removes != -1 {
		removes = strconv.Itoa(int(tm.Removes))
	}
	return &fde.Transaction{Id: fmt.Sprint(t.Moment), Date: d, AsOf: m, Debits: deb, Credits: cre,
			Memo: tm.Memo, Tags: tm.Tags, User: tm.User, Removes: removes},
		nil
}

func (s *integrations) fdeTransactionToDebTransaction(t *fde.Transaction, moment deb.Moment) (*deb.Transaction, error) {
	removes := -1
	if t.Removes != "" {
		var err error
		removes, err = strconv.Atoi(t.Removes)
		if err != nil {
			return nil, err
		}
	}
	metadata := transactionMetadata{t.Memo, nil, "", int64(removes)}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(metadata); err != nil {
		return nil, err
	}
	entries := deb.Entries{}
	id2idx := map[string]deb.Account{}
	var ids []string
	for _, e := range t.Debits {
		ids = append(ids, e.Account)
	}
	for _, e := range t.Credits {
		ids = append(ids, e.Account)
	}
	idxs, err := s.cr.Indexes(*s.coaid, ids, nil)
	if err != nil {
		return nil, err
	}
	for i, idx := range idxs {
		id2idx[ids[i]] = deb.Account(idx)
	}
	for _, e := range t.Debits {
		entries[id2idx[e.Account]+1] += e.Value
	}
	for _, e := range t.Credits {
		entries[id2idx[e.Account]+1] += -e.Value
	}
	return &deb.Transaction{
			Moment:   moment,
			Date:     deb.DateFromTime(t.Date),
			Entries:  entries,
			Metadata: buf.Bytes()},
		nil
}
