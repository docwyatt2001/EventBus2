package EventBus2

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil {
		t.Log("New EventBus not created!")
		t.Fail()
	}
}

func TestHasCallback(t *testing.T) {
	bus := New()
	err := bus.Subscribe("topic", func() {})
	if err != nil {
		t.Fail()
	}
	if bus.HasCallback("topic_topic") {
		t.Fail()
	}
	if !bus.HasCallback("topic") {
		t.Fail()
	}
}

func TestSubscribe(t *testing.T) {
	bus := New()
	if bus.Subscribe("topic", func() {}) != nil {
		t.Fail()
	}
	if bus.Subscribe("topic", "String") == nil {
		t.Fail()
	}
}

func TestSubscribeOnce(t *testing.T) {
	bus := New()
	if bus.SubscribeOnce("topic", func() {}) != nil {
		t.Fail()
	}
	if bus.SubscribeOnce("topic", "String") == nil {
		t.Fail()
	}
}

func TestSubscribeOnceAndManySubscribe(t *testing.T) {
	bus := New()
	event := "topic"
	flag := 0
	fn := func() { flag += 1 }
	err := bus.SubscribeOnce(event, fn)
	if err != nil {
		t.Fail()
	}
	err = bus.Subscribe(event, fn)
	if err != nil {
		t.Fail()
	}
	err = bus.Subscribe(event, fn)
	if err != nil {
		t.Fail()
	}
	err = bus.SubscribeOnce(event, fn)
	if err != nil {
		t.Fail()
	}
	bus.Publish(event)

	if flag != 4 {
		t.Fail()
	}
	bus.Publish(event)
	if flag != 6 {
		t.Fail()
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	handler := func() {}
	err := bus.Subscribe("topic", handler)
	if err != nil {
		t.Fail()
	}
	if bus.Unsubscribe("topic", handler) != nil {
		t.Fail()
	}
	if bus.Unsubscribe("topic", handler) == nil {
		t.Fail()
	}
}

type handler struct {
	val int
}

func (h *handler) Handle() {
	h.val++
}

func TestUnsubscribeMethod(t *testing.T) {
	bus := New()
	h := &handler{val: 0}

	err := bus.Subscribe("topic", h.Handle)
	if err != nil {
		t.Fail()
	}
	bus.Publish("topic")
	if bus.Unsubscribe("topic", h.Handle) != nil {
		t.Fail()
	}
	if bus.Unsubscribe("topic", h.Handle) == nil {
		t.Fail()
	}
	wg := bus.PublishWaitAsync("topic")
	bus.WaitAsync(wg)

	if h.val != 1 {
		t.Fail()
	}
}

func TestPublish(t *testing.T) {
	bus := New()
	err := bus.Subscribe("topic", func(a int, err error) {
		if a != 10 {
			t.Fail()
		}

		if err != nil {
			t.Fail()
		}
	})
	if err != nil {
		t.Fail()
	}
	bus.Publish("topic", 10, nil)
}

func TestSubscribeOnceAsync(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	err := bus.SubscribeOnceAsync("topic", func(a int, out *[]int) {
		*out = append(*out, a)
	})
	if err != nil {
		t.Fail()
	}

	bus.Publish("topic", 10, &results)
	wg := bus.PublishWaitAsync("topic", 10, &results)

	bus.WaitAsync(wg)

	if len(results) != 1 {
		t.Fail()
	}

	if bus.HasCallback("topic") {
		t.Fail()
	}
}

func TestMultiSubscribeOnceAsync(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	err := bus.SubscribeOnceAsync("topic", func(a int, out *[]int) {
		*out = append(*out, a)
	})
	if err != nil {
		t.Fail()
	}
	err = bus.SubscribeOnceAsync("topic", func(a int, out *[]int) {
		*out = append(*out, a)
	})
	if err != nil {
		t.Fail()
	}

	bus.Publish("topic", 10, &results)
	wg := bus.PublishWaitAsync("topic", 10, &results)

	bus.WaitAsync(wg)

	if len(results) != 2 {
		t.Fail()
	}

	if bus.HasCallback("topic") {
		t.Fail()
	}
}
func TestSubscribeAsyncTransactional(t *testing.T) {
	results := make([]int, 0)

	bus := New()
	err := bus.SubscribeAsync("topic", func(a int, out *[]int, dur string) {
		sleep, _ := time.ParseDuration(dur)
		time.Sleep(sleep)
		*out = append(*out, a)
	}, true)
	if err != nil {
		t.Fail()
	}

	bus.Publish("topic", 1, &results, "1s")
	wg := bus.PublishWaitAsync("topic", 2, &results, "0s")

	bus.WaitAsync(wg)

	if len(results) != 2 {
		t.Fail()
	}

	if results[0] != 1 || results[1] != 2 {
		t.Fail()
	}
}

func TestSubscribeAsync(t *testing.T) {
	results := make(chan int)

	bus := New()
	err := bus.SubscribeAsync("topic", func(a int, out chan<- int) {
		out <- a
	}, false)
	if err != nil {
		t.Fail()
	}

	bus.Publish("topic", 1, results)
	wg := bus.PublishWaitAsync("topic", 2, results)

	numResults := 0

	go func() {
		for range results {
			numResults++
		}
	}()

	bus.WaitAsync(wg)
	println(2)

	time.Sleep(10 * time.Millisecond)

	// todo race detected during execution of test
	//if numResults != 2 {
	//	t.Fail()
	//}
}

func TestSubscribeAsyncWithMultipleGoroutine(t *testing.T) {
	for i := 0; i < 100; i++ {

		bus := New()
		err := bus.SubscribeAsync("topic", func() {
			time.Sleep(time.Millisecond)
		}, false)

		if err != nil {
			t.Fail()
		}

		wg := bus.PublishWaitAsync("topic")

		go func() {
			time.Sleep(time.Millisecond)
			bus.Publish("topic")
		}()

		bus.WaitAsync(wg)
	}
}
