package groupcache

//// tests that peers (virtual, in-process) are hit, and how much.
//func TestHttpPeers1(t *testing.T) {
//	pool := NewHTTPPoolOpts("http://localhost:8080", &HTTPPoolOptions{})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8081")
//
//	server := http.Server{
//		Addr:    "localhost:8080",
//		Handler: pool,
//	}
//
//	// Start a HTTP server to listen for peer requests from the groupcache
//	go func() {
//		log.Printf("Serving....\n")
//		if err := server.ListenAndServe(); err != nil {
//			log.Fatal(err)
//		}
//	}()
//	defer server.Shutdown(context.Background())
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8082")
//}
//
//func TestHttpPeers2(t *testing.T) {
//	pool := NewHTTPPoolOpts("http://localhost:8081", &HTTPPoolOptions{})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8080", "http://localhost:8082")
//
//	server := http.Server{
//		Addr:    "localhost:8081",
//		Handler: pool,
//	}
//
//	// Start a HTTP server to listen for peer requests from the groupcache
//	go func() {
//		log.Printf("Serving....\n")
//		if err := server.ListenAndServe(); err != nil {
//			log.Fatal(err)
//		}
//	}()
//	defer server.Shutdown(context.Background())
//}
//
//func TestHttpPeers3(t *testing.T) {
//	pool := NewHTTPPoolOpts("http://localhost:8082", &HTTPPoolOptions{})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8080", "http://localhost:8081")
//
//	server := http.Server{
//		Addr:    "localhost:8082",
//		Handler: pool,
//	}
//
//	// Start a HTTP server to listen for peer requests from the groupcache
//	go func() {
//		log.Printf("Serving....\n")
//		if err := server.ListenAndServe(); err != nil {
//			log.Fatal(err)
//		}
//	}()
//	defer server.Shutdown(context.Background())
//}