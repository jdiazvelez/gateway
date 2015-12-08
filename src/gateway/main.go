package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gateway/admin"
	"gateway/config"
	"gateway/errors/report"
	"gateway/license"
	"gateway/logreport"
	"gateway/model"
	"gateway/proxy"
	"gateway/service"
	"gateway/soap"
	"gateway/sql"
	"gateway/version"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	if versionCheck() {
		fmt.Printf("Gateway %s (%s)\n",
			version.Name(), version.Commit())
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	// Setup logging
	log.SetFlags(log.Ldate | log.Lmicroseconds)
	log.SetOutput(admin.Interceptor)

	// Parse configuration
	conf, err := config.Parse(os.Args[1:])
	if err != nil {
		logreport.Fatalf("%s Error parsing config file: %v", config.System, err)
	}

	logreport.Printf("%s Running Gateway %s (%s)",
		config.System, version.Name(), version.Commit())

	// Set up error reporting
	if conf.Airbrake.APIKey != "" && conf.Airbrake.ProjectID != 0 && !conf.DevMode() {
		abEnv := "production"
		if conf.Airbrake.Environment != "" {
			abEnv = conf.Airbrake.Environment
		}
		report.RegisterReporter(report.ConfigureAirbrake(conf.Airbrake.APIKey, conf.Airbrake.ProjectID, abEnv))
	}

	// Setup the database
	db, err := sql.Connect(conf.Database)
	if err != nil {
		logreport.Fatalf("%s Error connecting to database: %v", config.System, err)
	}

	// Require a valid license key
	license.ValidateForever(conf.License, time.Hour)

	//check for sneaky people
	if license.DeveloperVersion {
		logreport.Printf("%s Checking developer version license constraints", config.System)
		accounts, _ := model.AllAccounts(db)
		if len(accounts) > license.DeveloperVersionAccounts {
			logreport.Fatalf("Developer version allows %v account(s).", license.DeveloperVersionAccounts)
		}
		for _, account := range accounts {
			var count int
			db.Get(&count, db.SQL("users/count"), account.ID)
			if count > license.DeveloperVersionUsers {
				logreport.Fatalf("Developer version allows %v user(s).", license.DeveloperVersionUsers)
			}

			apis, _ := model.AllAPIsForAccountID(db, account.ID)
			if len(apis) > license.DeveloperVersionAPIs {
				logreport.Fatalf("Developer version allows %v api(s).", license.DeveloperVersionAPIs)
			}
			for _, api := range apis {
				var count int
				db.Get(&count, db.SQL("proxy_endpoints/count_active"), api.ID)
				if count > license.DeveloperVersionProxyEndpoints {
					logreport.Fatalf("Developer version allows %v active proxy endpoint(s).", license.DeveloperVersionProxyEndpoints)
				}
			}
		}
	}

	if !db.UpToDate() {
		if conf.Database.Migrate || conf.DevMode() {
			if err = db.Migrate(); err != nil {
				logreport.Fatalf("Error migrating database: %v", err)
			}
		} else {
			logreport.Fatalf("%s The database is not up to date. "+
				"Please migrate by invoking with the -db-migrate flag.",
				config.System)
		}
	}

	// Set up dev mode account
	if conf.DevMode() {
		if _, err := model.FirstAccount(db); err != nil {
			logreport.Printf("%s Creating development account", config.System)
			if err := createDevAccount(db); err != nil {
				logreport.Fatalf("Could not create account: %v", err)
			}
		}
		if account, err := model.FirstAccount(db); err == nil {
			if users, _ := model.AllUsersForAccountID(db, account.ID); len(users) == 0 {
				logreport.Printf("%s Creating development user", config.System)
				if err := createDevUser(db); err != nil {
					logreport.Fatalf("Could not create account: %v", err)
				}
			}
		} else {
			logreport.Fatal("Dev account doesn't exist")
		}
	}

	service.ElasticLoggingService(conf.Elastic)
	service.BleveLoggingService(conf.Bleve)
	service.LogPublishingService(conf.Admin)

	model.InitializeRemoteEndpointTypes(conf.RemoteEndpoint)

	// Write script remote endpoints to tmp fireLifecycleHooks
	err = model.WriteAllScriptFiles(db)
	if err != nil {
		logreport.Printf("%s Unable to write script files due to error: %v", config.System, err)
	}

	// Configure SOAP
	err = soap.Configure(conf.Soap, conf.DevMode())
	if err != nil {
		logreport.Printf("%s Unable to configure SOAP due to error: %v.  SOAP services will not be available.", config.System, err)
	}

	// Cache all Jar files locally for quick access
	if err := model.CacheAllJarFiles(db); err != nil {
		logreport.Printf("%s Unable to cache SOAP remote endpoint jars on file system: %v", config.System, err)
	}

	// Start up listeners for soap_remote_endpoints, so that we can keep the file system in sync with the DB
	model.StartSoapRemoteEndpointUpdateListener(db)

	// Start the proxy
	logreport.Printf("%s Starting server", config.System)
	proxy := proxy.NewServer(conf, db)
	go proxy.Run()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		err := soap.Shutdown(sig)
		if err != nil {
			logreport.Printf("Error shutting down SOAP service: %v", err)
		}
		done <- true
	}()

	<-done

	logreport.Println("Shutdown complete")
}

func versionCheck() bool {
	return len(os.Args) >= 2 &&
		strings.ToLower(os.Args[1:2][0]) == "-version"
}

func createDevAccount(db *sql.DB) error {
	devAccount := &model.Account{Name: "Dev Account"}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err = devAccount.Insert(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

var symbols = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomPassword() string {
	password := make([]rune, 16)
	for i := range password {
		password[i] = symbols[rand.Intn(len(symbols))]
	}
	return string(password)
}

func createDevUser(db *sql.DB) error {
	account, err := model.FirstAccount(db)
	if err != nil {
		return err
	}
	user := &model.User{
		AccountID:   account.ID,
		Name:        "developer",
		Email:       "developer@justapis.com",
		NewPassword: randomPassword(),
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err = user.Insert(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
