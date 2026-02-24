package db

import "context"

func (p *Postgres) Ping(ctx context.Context) error {
	err := p.dbPool.Ping(ctx)
	if err != nil {
		return err
	}

	return nil
}
