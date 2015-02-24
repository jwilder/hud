package metrics

import (
	"time"
	log "github.com/Sirupsen/logrus"

	influxClient "github.com/influxdb/influxdb/client"
)

type InfluxDB struct {
	addr string
	user string
	pass string
	db   string
}

func NewInfluxDB(user, pass, addr, db string) (*InfluxDB, error) {
	return &InfluxDB{
		user: user,
		pass: pass,
		addr: addr,
		db:   db,
	}, nil
}
func (w *InfluxDB) SendForever(metrics chan *Collection) {

	config := &influxClient.ClientConfig{
		Host:     w.addr,
		Database: w.db,
	}

	if w.user != "" {
		config.Username = w.user
		config.Password = w.pass
	}

	client, err := influxClient.NewClient(config)

	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	for {

	RETRY:
		err = client.Ping()
		if err != nil {
			log.Errorf("ERROR: %s", err)
			time.Sleep(10 * time.Second)
			goto RETRY

		}

		err = client.AuthenticateDatabaseUser(config.Database, config.Username, config.Password)
		if err != nil {
			log.Errorf("ERROR: Unable to connect to influxdb: %s", err)
			time.Sleep(10 * time.Second)
			goto RETRY
		}

		series := []*influxClient.Series{}

		for {
			col := <-metrics

			now := time.Now().Unix()

			columns := []string{"time", "value"}

			for k, metric := range col.Metrics() {
				v := metric.Value()
				serie := &influxClient.Series{
					Name:    k,
					Columns: columns,
				}

				var dp []interface{}
				switch v := v.(type) {
				case int64:
					dp = []interface{}{now, v}
				case float64:
					dp = []interface{}{now, v}

				}
				serie.Points = [][]interface{}{dp}
				series = append(series, serie)
			}

			err := client.WriteSeriesWithTimePrecision(series, influxClient.Second)
			if err != nil {
				log.Errorf("ERROR: %s", err)
				time.Sleep(10 * time.Second)
				goto RETRY
			}
		}
	}
}
