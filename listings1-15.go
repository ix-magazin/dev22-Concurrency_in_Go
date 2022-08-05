////////////////////////////////////
//Listing 1: Programmabbruch in Go//
////////////////////////////////////

func main() {
	panic("Hello panic!")
}

// Output:
// panic: Hello panic!
// 
// goroutine 1 [running]:
// main.main()
// /tmp/sandbox1748122117/prog.go:8 +0x27
// 
// Program exited.

-------

//////////////////////////////////////////////////
//Listing 2: Einfache Definition einer Goroutine//
//////////////////////////////////////////////////

func routine(out chan string) {
	out <- "Hallo Goroutine!"
}

func main() {
	msg := make(chan string)
	go routine(msg)
	m := <-msg
	fmt.Println(m)
}

-------

///////////////////////
//Listing 3: Deadlock//
///////////////////////

func main() {
	msg := make(chan string)
	go routine(msg)
	m := <-msg
	fmt.Println(m)
	m = <-msg
}

// Hallo Goroutine!
// fatal error: all goroutines are asleep - deadlock!

-------

////////////////////////////
//Listing 4: Keine Ausgabe//
////////////////////////////

func routine() {
	fmt.Println("Hallo zusammen")
}

func main() {
	go routine()
}

-------

////////////////////////
//Listing 5: Data Race//
////////////////////////

func main() {
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			fmt.Println(i)
			wg.Done()
		}()
	}
	wg.Wait()
}
// Output:
// 5
// 5
// 5
// 5
// 5

-------

/////////////////////////////
//Listing 6: Fan-In Pattern//
/////////////////////////////

func main() {
	inc := make(chan int)
	result := make(chan int)
    // Worker der auf Events lauscht
	go func() { 
		counter := 0
		for n := range inc {
			counter += n
		}
		result <- counter
	}()
	wg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			inc <- 1
			wg.Done()
		}()
	}
	wg.Wait()
    // Channel schlieÃŸen, wenn keine Daten mehr gesendet werden
	close(inc) 
	fmt.Println(<-result)
}

-------

///////////////////////////////////////////////
//Listing 7: Beliebige Channel zusammenfassen//
///////////////////////////////////////////////

func fanIn(cs ...chan int) chan int {
	wg := sync.WaitGroup{}
	out := make(chan int)
	for _, c := range cs {
		wg.Add(1)
		go func(in chan int) {
			for n := range in {
				out <- n
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

-------

////////////////////////////////////////////////
//Listing 8: Grundstruktur für den HTTP-Server//
////////////////////////////////////////////////

type server struct {
	router  *http.ServeMux
	server  *http.Server
	counter int
}

func main() {
	addr := ":8080"
	s := &server{
		router: http.NewServeMux(),
		server: &http.Server{
			Addr:           addr,
			ReadTimeout:    3 * time.Second,
			WriteTimeout:   3 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}
	s.router.HandleFunc("/", s.handleCount())
	s.server.Handler = s.router
	fmt.Printf("Server started at %s\n", s.server.Addr)
	s.server.ListenAndServe()
}

-------

//////////////////////////////////////////////////
//Listing 9: Die Handle-Funktion für den Counter//
//////////////////////////////////////////////////

func (s *server) handleCount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.counter++
		io.WriteString(w, fmt.Sprintf("Counter: %03d", s.counter))
	}
}

-------

////////////////////////////////////////////////////
//Listing 10: Test des Handlers auf eine Data Race//
////////////////////////////////////////////////////

func TestServerRace(t *testing.T) {
	ts := &server{}
	handler := ts.handleCount()
	getCounter := func(out chan []byte) {
		w := httptest.NewRecorder()
		handler(w, &http.Request{})
		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		out <- body
	}
	out := make(chan []byte)
	go getCounter(out)
	go getCounter(out)
	<-out
	<-out
	want := 2
	if ts.counter != want {
		t.Errorf("Got: %d - Want: %d", ts.counter, want)
	}
}

-------

///////////////////////////////////////
//Listing 11: Test entdeckt Data Race//
///////////////////////////////////////

--- FAIL: TestServerRace (0.00s)
    testing.go:1312: race detected during execution of test
FAIL
exit status 1
FAIL    heise/concserver/raceserver     0.287s

-------

///////////////////////////////////////////
//Listing 12: Server nach dem Refactoring//
///////////////////////////////////////////

type server struct {
	router  *http.ServeMux
	server  *http.Server
	counter int
	inc     chan chan int
}

func (s *server) runCounter() {
	go func() {
		for resultChan := range s.inc {
			s.counter++
			resultChan <- s.counter
		}
	}()
}

func (s *server) handleCount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resultChan := make(chan int)
		s.inc <- resultChan
		io.WriteString(w, fmt.Sprintf("Counter: %03d", <-resultChan))
	}
}

-------

////////////////////////////////////////////////////////////
//Listing 13: Eigene Funktion für Kommunikation mit Server//
////////////////////////////////////////////////////////////

func (s *server) incCounter() int {
	resultChan := make(chan int)
	s.inc <- resultChan
	return <-resultChan
}

-------

//////////////////////////////////////
//Listing 14: Verwendung eines mutex//
//////////////////////////////////////

type server struct {
	router  *http.ServeMux
	server  *http.Server
	counter int
	mtx     *sync.RWMutex
}

func (s *server) incCounter() int {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.counter++
	return s.counter
}

-------

/////////////////////////////////////////////////////
//Listing 15: Sicherer Counter mit dem atomic-Paket//
/////////////////////////////////////////////////////

func (s *server) incCounter() int32 {
	return atomic.AddInt32(&s.counter, 1)
}