package model_test

import (
	"encoding/json"
	"fmt"

	"gateway/db"
	"gateway/db/mongo"
	"gateway/db/sql"
	"gateway/model"
	re "gateway/model/remote_endpoint"

	"github.com/jmoiron/sqlx/types"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func data() map[string]interface{} {
	return map[string]interface{}{
		"sqls-simple": map[string]interface{}{
			"config": map[string]interface{}{
				"server":   "some.url.net",
				"port":     1234,
				"user id":  "user",
				"password": "pass",
				"database": "db",
				"schema":   "dbschema",
			},
		},
		"sqls-complicated": map[string]interface{}{
			"config": map[string]interface{}{
				"server":             "some.url.net",
				"port":               1234,
				"user id":            "user",
				"password":           "pass",
				"database":           "db",
				"schema":             "dbschema",
				"connection timeout": 30,
			},
			"maxOpenConn": 80,
			"maxIdleConn": 100,
		},
		"sqls-badConfig": map[string]interface{}{
			"config": map[string]interface{}{
				"server": "some.url.net",
			},
		},
		"sqls-badConfigType": map[string]interface{}{
			"config": 8,
		},
		"sqls-badMaxIdleType": map[string]interface{}{
			"config": map[string]interface{}{
				"server":   "some.url.net",
				"port":     1234,
				"user id":  "user",
				"password": "pass",
				"database": "db",
				"schema":   "dbschema",
			},
			"maxOpenConn": "hello",
		},
		"mysql-simple": map[string]interface{}{
			"config": map[string]interface{}{
				"server":   "some.url.net",
				"port":     1234,
				"username": "user",
				"password": "pass",
				"dbname":   "db",
				"schema":   "dbschema",
			},
		},
		"mysql-complicated": map[string]interface{}{
			"config": map[string]interface{}{
				"server":             "some.url.net",
				"port":               1234,
				"username":           "user",
				"password":           "pass",
				"dbname":             "db",
				"schema":             "dbschema",
				"connection timeout": 30,
			},
			"maxOpenConn": 80,
			"maxIdleConn": 100,
		},
		"mysql-badConfig": map[string]interface{}{
			"config": map[string]interface{}{
				"server": "some.url.net",
			},
		},
		"mysql-badConfigType": map[string]interface{}{
			"config": 8,
		},
		"mysql-badMaxIdleType": map[string]interface{}{
			"config": map[string]interface{}{
				"server":   "some.url.net",
				"port":     1234,
				"username": "user",
				"password": "pass",
				"dbname":   "db",
				"schema":   "dbschema",
			},
			"maxOpenConn": "hello",
		},
		"pq-simple": map[string]interface{}{
			"config": map[string]interface{}{
				"host":     "some.url.net",
				"port":     1234,
				"user":     "user",
				"password": "pass",
				"dbname":   "db",
				"sslmode":  "prefer",
			},
		},
		"pq-complicated": map[string]interface{}{
			"config": map[string]interface{}{
				"host":     "some.url.net",
				"port":     1234,
				"user":     "user",
				"password": "pass",
				"dbname":   "db",
				"sslmode":  "prefer",
			},
			"maxOpenConn": 80,
			"maxIdleConn": 100,
		},
		"pq-badConfig": map[string]interface{}{
			"config": map[string]interface{}{},
		},
		"pq-badConfigType": map[string]interface{}{
			"config": 8,
		},
		"pq-badMaxIdleType": map[string]interface{}{
			"config": map[string]interface{}{
				"host":     "some.url.net",
				"port":     1234,
				"user":     "user",
				"password": "pass",
				"dbname":   "db",
			},
			"maxOpenConn": "hello",
		},
		"mongo-complicated": map[string]interface{}{
			"config": map[string]interface{}{
				"hosts": []interface{}{
					map[string]interface{}{
						"host": "test.com",
						"port": float64(123),
					},
				},
				"username": "user",
				"password": "pass",
				"database": "db",
			},
			"limit": 123,
		},
	}
}

func specs() map[string]db.Specifier {
	specs := make(map[string]db.Specifier)
	for _, which := range []struct {
		name string
		kind string
	}{
		{"sqls-simple", model.RemoteEndpointTypeSQLServer},
		{"sqls-complicated", model.RemoteEndpointTypeSQLServer},
		{"pq-simple", model.RemoteEndpointTypePostgres},
		{"pq-complicated", model.RemoteEndpointTypePostgres},
		{"mysql-simple", model.RemoteEndpointTypeMySQL},
		{"mysql-complicated", model.RemoteEndpointTypeMySQL},
		{"mongo-complicated", model.RemoteEndpointTypeMongo},
	} {
		d := data()[which.name].(map[string]interface{})
		js, err := json.Marshal(d)
		if err != nil {
			panic(err)
		}
		var s db.Specifier
		switch which.kind {
		case model.RemoteEndpointTypeSQLServer:
			var conf re.SQLServer
			err = json.Unmarshal(js, &conf)
			if err != nil {
				panic(err)
			}
			s, err = sql.Config(
				sql.Connection(conf.Config),
				sql.MaxOpenIdle(conf.MaxOpenConn, conf.MaxIdleConn),
			)
		case model.RemoteEndpointTypePostgres:
			var conf re.Postgres
			err = json.Unmarshal(js, &conf)
			if err != nil {
				panic(err)
			}
			s, err = sql.Config(
				sql.Connection(conf.Config),
				sql.MaxOpenIdle(conf.MaxOpenConn, conf.MaxIdleConn),
			)
		case model.RemoteEndpointTypeMySQL:
			var conf re.MySQL
			err = json.Unmarshal(js, &conf)
			if err != nil {
				panic(err)
			}
			s, err = sql.Config(
				sql.Connection(conf.Config),
				sql.MaxOpenIdle(conf.MaxOpenConn, conf.MaxIdleConn),
			)
		case model.RemoteEndpointTypeMongo:
			var conf re.Mongo
			err = json.Unmarshal(js, &conf)
			if err != nil {
				panic(err)
			}
			s, err = mongo.Config(
				mongo.Connection(conf.Config),
				mongo.PoolLimit(conf.Limit),
			)
		default:
			err = fmt.Errorf("no such type %q", which.kind)
		}
		if err != nil {
			panic(err)
		}
		specs[which.name] = s
	}
	return specs
}

func (s *ModelSuite) TestDBConfig(c *gc.C) {
	for i, t := range []struct {
		should      string
		givenConfig string
		givenType   string
		expectSpec  string
		expectError string
	}{{
		should:      "(SQLS) work with a simple config",
		givenConfig: "sqls-simple",
		givenType:   model.RemoteEndpointTypeSQLServer,
		expectSpec:  "sqls-simple",
	}, {
		should:      "(SQLS) work with a complex config",
		givenConfig: "sqls-complicated",
		givenType:   model.RemoteEndpointTypeSQLServer,
		expectSpec:  "sqls-complicated",
	}, {
		should:      "(SQLS) fail with a bad config",
		givenConfig: "sqls-badConfig",
		givenType:   model.RemoteEndpointTypeSQLServer,
		expectError: `mssql config errors: ` +
			`bad value "" for "user id"; ` +
			`bad value "" for "password"; ` +
			`bad value "" for "database"`,
	}, {
		should:      "(SQLS) fail with a bad config type",
		givenConfig: "sqls-badConfigType",
		givenType:   model.RemoteEndpointTypeSQLServer,
		expectError: `bad JSON for SQL Server config: ` +
			`json: cannot unmarshal number into Go value of type sql.SQLServerSpec`,
	}, {
		should:      "(SQLS) fail with a bad max idle type",
		givenConfig: "sqls-badMaxIdleType",
		givenType:   model.RemoteEndpointTypeSQLServer,
		expectError: `bad JSON for SQL Server config: ` +
			`json: cannot unmarshal string into Go value of type int`,
	}, {
		should:      "(MySQL) work with a simple config",
		givenConfig: "mysql-simple",
		givenType:   model.RemoteEndpointTypeMySQL,
		expectSpec:  "mysql-simple",
	}, {
		should:      "(MySQL) work with a complex config",
		givenConfig: "mysql-complicated",
		givenType:   model.RemoteEndpointTypeMySQL,
		expectSpec:  "mysql-complicated",
	}, {
		should:      "(MySQL) fail with a bad config",
		givenConfig: "mysql-badConfig",
		givenType:   model.RemoteEndpointTypeMySQL,
		expectError: `mysql config errors: ` +
			`bad value "" for "username"; ` +
			`bad value "" for "password"; ` +
			`bad value "" for "dbname"`,
	}, {
		should:      "(MySQL) fail with a bad config type",
		givenConfig: "mysql-badConfigType",
		givenType:   model.RemoteEndpointTypeMySQL,
		expectError: `bad JSON for MySQL config: ` +
			`json: cannot unmarshal number into Go value of type sql.MySQLSpec`,
	}, {
		should:      "(MySQL) fail with a bad max idle type",
		givenConfig: "mysql-badMaxIdleType",
		givenType:   model.RemoteEndpointTypeMySQL,
		expectError: `bad JSON for MySQL config: ` +
			`json: cannot unmarshal string into Go value of type int`,
	}, {
		should:      "(PSQL) work with a simple config",
		givenConfig: "pq-simple",
		givenType:   model.RemoteEndpointTypePostgres,
		expectSpec:  "pq-simple",
	}, {
		should:      "(PSQL) work with a complex config",
		givenConfig: "pq-complicated",
		givenType:   model.RemoteEndpointTypePostgres,
		expectSpec:  "pq-complicated",
	}, {
		should:      "(PSQL) fail with a bad config",
		givenConfig: "pq-badConfig",
		givenType:   model.RemoteEndpointTypePostgres,
		expectError: `pgx config errors: ` +
			`bad value "" for "user"; ` +
			`bad value "" for "password"; ` +
			`bad value "" for "dbname"; ` +
			`bad value "" for "host"; ` +
			`bad value "" for "sslmode"`,
	}, {
		should:      "(PSQL) fail with a bad config type",
		givenConfig: "pq-badConfigType",
		givenType:   model.RemoteEndpointTypePostgres,
		expectError: `bad JSON for Postgres config: ` +
			`json: cannot unmarshal number into Go value of type sql.PostgresSpec`,
	}, {
		should:      "(PSQL) fail with a bad max idle type",
		givenConfig: "pq-badMaxIdleType",
		givenType:   model.RemoteEndpointTypePostgres,
		expectError: `bad JSON for Postgres config: ` +
			`json: cannot unmarshal string into Go value of type int`,
	}, {
		should:      "Mongo work with a complex config",
		givenConfig: "mongo-complicated",
		givenType:   model.RemoteEndpointTypeMongo,
		expectSpec:  "mongo-complicated",
	}} {
		c.Logf("Test %d: should %s", i, t.should)
		data := data()[t.givenConfig]
		dataJSON, err := json.Marshal(data)
		endpoint := &model.RemoteEndpoint{
			Type: t.givenType,
			Data: types.JsonText(json.RawMessage(dataJSON)),
		}
		spec, err := endpoint.DBConfig()
		if t.expectError != "" {
			c.Check(err, gc.ErrorMatches, t.expectError)
			continue
		}
		c.Assert(err, jc.ErrorIsNil)
		expectSpec := specs()[t.expectSpec]
		c.Check(spec, jc.DeepEquals, expectSpec)
	}
}
