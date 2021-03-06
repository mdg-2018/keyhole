// Copyright 2018 Kuei-chun Chen. All rights reserved.

package mdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"strings"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"golang.org/x/crypto/ssh/terminal"
)

// KeyholeDB default database
var KeyholeDB = "_KEYHOLE_88800"

// ExamplesCollection default test colection
var ExamplesCollection = "__examples"

// NewMongoClient new mongo client
func NewMongoClient(uri string, files ...string) (*mongo.Client, error) {
	var err error
	var client *mongo.Client
	var connString connstring.ConnString
	if connString, err = connstring.Parse(uri); err != nil {
		return client, err
	}
	opts := options.Client().ApplyURI(uri)
	if connString.ReplicaSet == "" && len(connString.Hosts) == 1 && strings.HasPrefix(uri, "mongodb://") {
		opts.SetDirect(true)
	}
	if connString.Username == "" {
		opts.Auth = nil
	}
	if len(files) > 0 && files[0] != "" {
		connString.SSL = true
		roots := x509.NewCertPool()
		var caBytes []byte
		if caBytes, err = ioutil.ReadFile(files[0]); err != nil {
			return nil, err
		}
		if ok := roots.AppendCertsFromPEM(caBytes); !ok {
			return client, errors.New("failed to parse root certificate")
		}
		var certs tls.Certificate
		if len(files) >= 2 && files[1] != "" {
			var clientBytes []byte
			if clientBytes, err = ioutil.ReadFile(files[1]); err != nil {
				return nil, err
			}
			if certs, err = tls.X509KeyPair(clientBytes, clientBytes); err != nil {
				return nil, err
			}
		}
		opts.SetTLSConfig(&tls.Config{RootCAs: roots, Certificates: []tls.Certificate{certs}})
	}
	if client, err = mongo.NewClient(opts); err != nil {
		return client, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = client.Connect(ctx); err != nil {
		panic(err)
	}
	err = client.Ping(ctx, nil)
	return client, err
}

// ParseURI checks if password is included
func ParseURI(uri string) (connstring.ConnString, error) {
	var err error
	var connString connstring.ConnString
	connString, err = connstring.Parse(uri)                     // ignore error to accomodate authMechanism=PLAIN
	if connString.Username != "" && connString.Password == "" { // missing password, prompt for it
		fmt.Print("Enter Password: ")
		var data []byte
		if data, err = terminal.ReadPassword(int(syscall.Stdin)); err != nil {
			return connString, err
		}
		fmt.Println("")
		connString.Password = string(data)
		i := strings.Index(uri, connString.Username) + len(connString.Username)
		uri = (uri)[:i] + ":" + template.URLQueryEscaper(connString.Password) + (uri)[i:]
		return connstring.Parse(uri)
	}
	return connString, err
}
