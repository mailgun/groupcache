package groupcache_test

import (
	"context"
	"fmt"
	"github.com/mailgun/groupcache/v2"
	"log"
	"time"
)

func ExampleUsage() {
	/*
		// Keep track of peers in our cluster and add our instance to the pool `http://localhost:8080`
		pool := groupcache.NewHTTPPoolOpts("http://localhost:8080", &groupcache.HTTPPoolOptions{})

		// Add more peers to the cluster
		//pool.Set("http://peer1:8080", "http://peer2:8080")

		server := http.Server{
			Addr:    "localhost:8080",
			Handler: pool,
		}

		// Start a HTTP server to listen for peer requests from the groupcache
		go func() {
			log.Printf("Serving....\n")
			if err := server.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
		}()
		defer server.Shutdown(context.Background())
	*/

	// Create a new group cache with a max cache size of 3MB
	group := groupcache.NewGroup("users", 3000000, groupcache.GetterFunc(
		func(ctx context.Context, id string, dest groupcache.Sink) error {

			// In a real scenario we might fetch the value from a database.
			/*if user, err := fetchUserFromMongo(ctx, id); err != nil {
				return err
			}*/

			user := User{
				Id:      "12345",
				Name:    "John Doe",
				Age:     40,
				IsSuper: true,
			}

			// Set the user in the groupcache to expire after 5 minutes
			if err := dest.SetProto(&user, time.Now().Add(time.Minute*5)); err != nil {
				return err
			}
			return nil
		},
	))

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := group.Get(ctx, "12345", groupcache.ProtoSink(&user)); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("-- User --\n")
	fmt.Printf("Id: %s\n", user.Id)
	fmt.Printf("Name: %s\n", user.Name)
	fmt.Printf("Age: %d\n", user.Age)
	fmt.Printf("IsSuper: %t\n", user.IsSuper)

	/*
		// Remove the key from the groupcache
		if err := group.Remove(ctx, "12345"); err != nil {
			fmt.Printf("Remove Err: %s\n", err)
			log.Fatal(err)
		}
	*/

	// Output: -- User --
	// Id: 12345
	// Name: John Doe
	// Age: 40
	// IsSuper: true
}


// tests that peers (virtual, in-process) are hit, and how much.
//func TestHttpPeers1(t *testing.T) {
//	pool := groupcache.NewHTTPPoolOpts("http://localhost:8080", &groupcache.HTTPPoolOptions{Replicas: 20})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8080","http://localhost:8081")
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
//
//	defer server.Shutdown(context.Background())
//
//	// Add more peers to the cluster
//	//pool.Set("http://localhost:8082")
//
//	group := groupcache.NewGroup("users", 3000000, groupcache.GetterFunc(
//		func(ctx context.Context, id string, dest groupcache.Sink) error {
//
//			// In a real scenario we might fetch the value from a database.
//			/*if user, err := fetchUserFromMongo(ctx, id); err != nil {
//				return err
//			}*/
//
//			user := User{
//				Id:      "12345",
//				Name:    "John Doe",
//				Age:     40,
//				IsSuper: true,
//			}
//
//			// Set the user in the groupcache to expire after 5 minutes
//			if err := dest.SetProto(&user, time.Now().Add(time.Minute*5)); err != nil {
//				return err
//			}
//			return nil
//		},
//	))
//
//	go func() {
//		http.HandleFunc("/add_node", func(writer http.ResponseWriter, request *http.Request) {
//			// TODO 添加节点
//			pool.Set("http://localhost:8081", "http://localhost:8080","http://localhost:8082","http://localhost:8083")
//			fmt.Fprint(writer, "add success")
//			return
//		})
//		http.HandleFunc("/get_user", func(writer http.ResponseWriter, request *http.Request) {
//			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
//			defer cancel()
//			var user User
//			if err := group.Get(ctx, "12345", groupcache.ProtoSink(&user)); err != nil {
//				log.Fatal(err)
//			}
//			fmt.Printf("-- User --\n")
//			fmt.Printf("Id: %s\n", user.Id)
//			fmt.Printf("Name: %s\n", user.Name)
//			fmt.Printf("Age: %d\n", user.Age)
//			fmt.Printf("IsSuper: %t\n", user.IsSuper)
//			fmt.Fprint(writer, user.Name)
//			return
//		})
//		http.ListenAndServe("localhost:8090", nil)
//	}()
//	for {
//		time.Sleep(10*time.Second)
//	}
//}
//
//func TestHttpPeers2(t *testing.T) {
//	pool := groupcache.NewHTTPPoolOpts("http://localhost:8081", &groupcache.HTTPPoolOptions{Replicas: 20})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8081", "http://localhost:8080")
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
//
//
//	group := groupcache.NewGroup("users", 3000000, groupcache.GetterFunc(
//		func(ctx context.Context, id string, dest groupcache.Sink) error {
//
//			// In a real scenario we might fetch the value from a database.
//			/*if user, err := fetchUserFromMongo(ctx, id); err != nil {
//				return err
//			}*/
//
//			user := User{
//				Id:      "12345",
//				Name:    "TestHttpPeers2",
//				Age:     40,
//				IsSuper: true,
//			}
//
//			// Set the user in the groupcache to expire after 5 minutes
//			if err := dest.SetProto(&user, time.Now().Add(time.Minute*5)); err != nil {
//				return err
//			}
//			return nil
//		},
//	))
//
//	group.Stats.LocalLoads.Get()
//
//	go func() {
//		http.HandleFunc("/add_node", func(writer http.ResponseWriter, request *http.Request) {
//			// TODO 添加节点
//			pool.Set("http://localhost:8081", "http://localhost:8080","http://localhost:8082","http://localhost:8083")
//			fmt.Fprint(writer, "add success")
//			return
//		})
//		http.HandleFunc("/get_user", func(writer http.ResponseWriter, request *http.Request) {
//			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
//			defer cancel()
//			var user User
//			if err := group.Get(ctx, "12345", groupcache.ProtoSink(&user)); err != nil {
//				log.Fatal(err)
//			}
//			fmt.Printf("-- User --\n")
//			fmt.Printf("Id: %s\n", user.Id)
//			fmt.Printf("Name: %s\n", user.Name)
//			fmt.Printf("Age: %d\n", user.Age)
//			fmt.Printf("IsSuper: %t\n", user.IsSuper)
//			fmt.Fprint(writer, user.Name)
//			return
//		})
//		http.ListenAndServe(":8091", nil)
//	}()
//	defer server.Shutdown(context.Background())
//
//	for {
//		time.Sleep(10*time.Second)
//	}
//}
//
//func TestHttpPeers3(t *testing.T) {
//	pool := groupcache.NewHTTPPoolOpts("http://localhost:8082", &groupcache.HTTPPoolOptions{Replicas: 20})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8082")
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
//
//	group := groupcache.NewGroup("users", 3000000, groupcache.GetterFunc(
//		func(ctx context.Context, id string, dest groupcache.Sink) error {
//
//			// In a real scenario we might fetch the value from a database.
//			/*if user, err := fetchUserFromMongo(ctx, id); err != nil {
//				return err
//			}*/
//
//			user := User{
//				Id:      "12345",
//				Name:    "TestHttpPeers3",
//				Age:     40,
//				IsSuper: true,
//			}
//
//			// Set the user in the groupcache to expire after 5 minutes
//			if err := dest.SetProto(&user, time.Now().Add(time.Minute*5)); err != nil {
//				return err
//			}
//			return nil
//		},
//	))
//	group.Stats.LocalLoads.Get()
//
//	go func() {
//		http.HandleFunc("/add_node", func(writer http.ResponseWriter, request *http.Request) {
//			// TODO 添加节点
//			pool.Set("http://localhost:8081", "http://localhost:8080","http://localhost:8082","http://localhost:8083")
//			fmt.Fprint(writer, "add success")
//			return
//		})
//
//		http.HandleFunc("/get_user", func(writer http.ResponseWriter, request *http.Request) {
//			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
//			defer cancel()
//			var user User
//			if err := group.Get(ctx, "12345", groupcache.ProtoSink(&user)); err != nil {
//				log.Fatal(err)
//			}
//			fmt.Printf("-- User --\n")
//			fmt.Printf("Id: %s\n", user.Id)
//			fmt.Printf("Name: %s\n", user.Name)
//			fmt.Printf("Age: %d\n", user.Age)
//			fmt.Printf("IsSuper: %t\n", user.IsSuper)
//			fmt.Fprint(writer, user.Name)
//			return
//		})
//		http.ListenAndServe(":8092", nil)
//	}()
//	defer server.Shutdown(context.Background())
//
//	for {
//		time.Sleep(10*time.Second)
//	}
//}
//
//func TestHttpPeers4(t *testing.T) {
//	pool := groupcache.NewHTTPPoolOpts("http://localhost:8083", &groupcache.HTTPPoolOptions{Replicas: 20})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8083")
//
//	server := http.Server{
//		Addr:    "localhost:8083",
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
//
//	group := groupcache.NewGroup("users", 3000000, groupcache.GetterFunc(
//		func(ctx context.Context, id string, dest groupcache.Sink) error {
//
//			// In a real scenario we might fetch the value from a database.
//			/*if user, err := fetchUserFromMongo(ctx, id); err != nil {
//				return err
//			}*/
//
//			user := User{
//				Id:      "12345",
//				Name:    "TestHttpPeers3",
//				Age:     40,
//				IsSuper: true,
//			}
//
//			// Set the user in the groupcache to expire after 5 minutes
//			if err := dest.SetProto(&user, time.Now().Add(time.Minute*5)); err != nil {
//				return err
//			}
//			return nil
//		},
//	))
//	group.Stats.LocalLoads.Get()
//
//	go func() {
//		http.HandleFunc("/add_node", func(writer http.ResponseWriter, request *http.Request) {
//			// TODO 添加节点
//			pool.Set("http://localhost:8081", "http://localhost:8080","http://localhost:8082","http://localhost:8083")
//			fmt.Fprint(writer, "add success")
//			return
//		})
//
//		http.HandleFunc("/get_user", func(writer http.ResponseWriter, request *http.Request) {
//			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
//			defer cancel()
//			var user User
//			if err := group.Get(ctx, "12345", groupcache.ProtoSink(&user)); err != nil {
//				log.Fatal(err)
//			}
//			fmt.Printf("-- User --\n")
//			fmt.Printf("Id: %s\n", user.Id)
//			fmt.Printf("Name: %s\n", user.Name)
//			fmt.Printf("Age: %d\n", user.Age)
//			fmt.Printf("IsSuper: %t\n", user.IsSuper)
//			fmt.Fprint(writer, user.Name)
//			return
//		})
//		http.ListenAndServe(":8093", nil)
//	}()
//	defer server.Shutdown(context.Background())
//
//	for {
//		time.Sleep(10*time.Second)
//	}
//}
//
//func TestGroupBatchGetFromPeers1(t *testing.T) {
//	pool := groupcache.NewHTTPPoolOpts("http://localhost:8081", &groupcache.HTTPPoolOptions{Replicas: 20})
//
//	// Add more peers to the cluster
//	pool.Set("http://localhost:8081","http://localhost:8082")
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
//
//	getter := func(_ context.Context, keyList []string, destList []groupcache.Sink) []error {
//		errList := make([]error,0)
//		for i := 0; i < len(destList); i++ {
//			indexStr := strconv.FormatInt(int64(i),10)
//			user := User{
//				Id:      indexStr,
//				Name:    keyList[i],
//				Age:     40 + int64(i),
//				IsSuper: true,
//			}
//
//			// Set the user in the groupcache to expire after 5 minutes
//			if err := destList[i].SetProto(&user, time.Now().Add(time.Minute*5)); err != nil {
//				errList = append(errList,err)
//			}
//		}
//		return errList
//	}
//	group := groupcache.NewGroup("users", 3000000, groupcache.BatchGetterFunc(getter))
//	group.Stats.LocalLoads.Get()
//
//	go func() {
//		http.HandleFunc("/add_node", func(writer http.ResponseWriter, request *http.Request) {
//			// TODO 添加节点
//			pool.Set("http://localhost:8081", "http://localhost:8080","http://localhost:8082","http://localhost:8083")
//			fmt.Fprint(writer, "add success")
//			return
//		})
//
//		http.HandleFunc("/get_user", func(writer http.ResponseWriter, request *http.Request) {
//			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
//			defer cancel()
//			var user User
//			if err := group.Get(ctx, "12345", groupcache.ProtoSink(&user)); err != nil {
//				log.Fatal(err)
//			}
//			fmt.Printf("-- User --\n")
//			fmt.Printf("Id: %s\n", user.Id)
//			fmt.Printf("Name: %s\n", user.Name)
//			fmt.Printf("Age: %d\n", user.Age)
//			fmt.Printf("IsSuper: %t\n", user.IsSuper)
//			fmt.Fprint(writer, user.Name)
//			return
//		})
//		http.ListenAndServe(":8091", nil)
//	}()
//	defer server.Shutdown(context.Background())
//
//	for {
//		time.Sleep(10*time.Second)
//	}
//}
