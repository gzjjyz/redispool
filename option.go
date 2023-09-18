package redispool

type Option func(p *Pool)

func WithHost(host string) Option {
	return func(p *Pool) {
		p.host = host
	}
}

func WithPassword(pwd string) Option {
	return func(p *Pool) {
		p.password = pwd
	}
}

func WithDB(db int) Option {
	return func(p *Pool) {
		p.db = db
	}
}

func WithMinCount(count int) Option {
	return func(p *Pool) {
		p.minCount = count
	}
}
