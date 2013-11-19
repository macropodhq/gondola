package orm

import (
	"strings"
	"testing"
	"time"
)

type BadEvent struct {
	Timestamp int `orm:",references=Timestamp"`
	Name      string
}

type Event struct {
	Id        int64 `orm:",primary_key,auto_increment"`
	Timestamp int64 `orm:",references=Timestamp"`
	Name      string
}

type TimedEvent struct {
	Id    int64 `orm:",primary_key,auto_increment"`
	Start int64 `orm:",references=Timestamp(Id)"` // This is the same that reference just Timestamp
	End   int64 `orm:",references=Timestamp"`
	Name  string
}

var (
	eventNames = []string{"E1", "E2", "E3"}
	eventCount = len(eventNames)
)

func testBadReferences(t *testing.T, o *Orm) {
	// TODO: Test for bad references which omit the field
	// and bad references to non-existant field names
	_, err := o.Register((*BadEvent)(nil), &Options{
		Table: "test_references_bad_event",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.Register((*Timestamp)(nil), &Options{
		Table: "test_references_bad_timestamp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Initialize(); err == nil || !strings.Contains(err.Error(), "type") {
		t.Errorf("expecting error when registering FK of different type, got %s instead", err)
	}
}

func testCount(t *testing.T, count int, expected int, msg string) {
	if expected < 0 {
		expected = eventCount
	}
	if count != expected {
		t.Errorf("expecting %d results for %s, got %d instead", expected, msg, count)
	}
}

func testEvent(t *testing.T, event *Event, pos int) {
	if pos < 0 {
		pos = len(eventNames) - pos
	}
	if expect := eventNames[pos]; expect != event.Name {
		t.Errorf("expecting event name %q, got %q instead", expect, event.Name)
	}
}

func testIterErr(t *testing.T, iter *Iter) {
	if err := iter.Err(); err != nil {
		t.Error(err)
	}
}

func testReferences(t *testing.T, o *Orm) {
	// Register Event first and then Timestamp. The ORM should
	// re-arrange them so Timestamp is created before Event.
	// TODO: Test for ambiguous joins, they don't work yet
	eventTable, err := o.Register((*Event)(nil), &Options{
		Table: "test_references_event",
	})
	if err != nil {
		t.Fatal(err)
	}
	timestampTable, err := o.Register((*Timestamp)(nil), &Options{
		Table:   "test_references_timestamp",
		Default: true,
		Name:    "Timestamp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := o.Initialize(); err != nil {
		t.Fatal(err)
	}
	// Insert a few objects
	t1 := time.Now().UTC()
	t2 := t1.Add(time.Hour)
	var timestamps []*Timestamp
	for _, v := range []time.Time{t1, t2} {
		ts := &Timestamp{Timestamp: v}
		if _, err := o.Insert(ts); err != nil {
			t.Fatal(err)
		}
		timestamps = append(timestamps, ts)
	}
	for _, v := range eventNames {
		if _, err := o.Insert(&Event{Timestamp: timestamps[0].Id, Name: v}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := o.Insert(&Event{Name: "E4"}); err != nil {
		t.Fatal(err)
	}
	var timestamp *Timestamp
	var event *Event
	// Ambiguous query, should return an error
	iter := o.Query(Eq("Id", 1)).Iter()
	for iter.Next(&timestamp, &event) {
	}
	if err := iter.Err(); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expecting ambiguous query error, got %v instead", err)
	}
	var count int
	// Fetch all the events for timestamp with id=1
	iter = o.Query(Eq("Timestamp|Id", 1)).Sort("Event|Id", ASC).Iter()
	for count = 0; iter.Next(&timestamp, &event); count++ {
		if !equalTimes(t1, timestamp.Timestamp) {
			t.Errorf("expecting time %v, got %v instead", t1, timestamp.Timestamp)
		}
		testEvent(t, event, count)
	}
	testCount(t, count, -1, "timestamp Id=1")
	testIterErr(t, iter)
	// Fetch all the events for timestamp with id=1, but ignore the timestamp
	iter = o.Query(Eq("Timestamp|Id", 1)).Sort("Event|Name", ASC).Iter()
	for count = 0; iter.Next((*Timestamp)(nil), &event); count++ {
		testEvent(t, event, count)
	}
	testCount(t, count, -1, "timestamp Id=1")
	testIterErr(t, iter)
	// This should produce an untyped nil pointer error
	iter = o.Query(Eq("Timestamp|Id", 1)).Iter()
	for iter.Next(nil, &event) {
	}
	if err := iter.Err(); err != errUntypedNilPointer {
		t.Errorf("expecting error %s, got %s instead", errUntypedNilPointer, err)
	}
	// Fetch all the events for timestamp with id=1, but ignore the timestamp using an
	// explicit table.
	iter = o.Query(Eq("Timestamp|Id", 1)).Sort("Event|Name", ASC).Table(timestampTable.Skip().MustJoin(eventTable, nil, InnerJoin)).Iter()
	for count = 0; iter.Next(nil, &event); count++ {
		testEvent(t, event, count)
	}
	testCount(t, count, -1, "timestamp Id=1")
	testIterErr(t, iter)
	// Fetch all the events for timestamp with id=2. There are no events so event
	// should be nil.
	iter = o.Query(Eq("Timestamp|Id", 2)).Join(LeftJoin).Iter()
	for count = 0; iter.Next(&timestamp, &event); count++ {
		if event != nil {
			t.Errorf("expecting nil event for Timestamp Id=2, got %+v instead", event)
		}
	}
	testCount(t, count, 1, "Timestamp Id=2")
	testIterErr(t, iter)
	// Fetch event with id=2 with its timestamp.
	iter = o.Query(Eq("Event|Id", 2)).Iter()
	for count = 0; iter.Next(&event, &timestamp); count++ {
		if event.Name != "E2" {
			t.Errorf("expecting event name E2, got %s instead", event.Name)
		}
		if !equalTimes(t1, timestamp.Timestamp) {
			t.Errorf("expecting time %v, got %v instead", t1, timestamp.Timestamp)
		}
	}
	testCount(t, count, 1, "Event Id=2")
	testIterErr(t, iter)
	// Now do the same but pass (timestamp, event) to next. The ORM
	// should perform the join correctly anyway.
	iter = o.Query(Eq("Event|Id", 2)).Iter()
	for count = 0; iter.Next(&timestamp, &event); count++ {
		if event.Name != "E2" {
			t.Errorf("expecting event name E2, got %s instead", event.Name)
		}
		if !equalTimes(t1, timestamp.Timestamp) {
			t.Errorf("expecting time %v, got %v instead", t1, timestamp.Timestamp)
		}
	}
	testCount(t, count, 1, "Event Id=2")
	testIterErr(t, iter)
	iter = o.Query(Eq("Event|Id", 4)).Table(eventTable.MustJoin(timestampTable, nil, LeftJoin)).Iter()
	for count = 0; iter.Next(&event, &timestamp); count++ {
		if event.Name != "E4" {
			t.Errorf("expecting event name E4, got %s instead", event.Name)
		}
		if timestamp != nil {
			t.Errorf("expecting nil Timestamp, got %v instead", timestamp)
		}
	}
	testCount(t, count, 1, "Event Id=4")
	testIterErr(t, iter)
}

func TestBadReferences(t *testing.T) {
	runTest(t, testBadReferences)
}

func TestReferences(t *testing.T) {
	runTest(t, testReferences)
}