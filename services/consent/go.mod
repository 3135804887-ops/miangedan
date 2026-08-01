module miangedan/services/consent

go 1.26

require (
	miangedan/services/identity v0.0.0
	miangedan/services/notify v0.0.0 // indirect
	miangedan/services/region v0.0.0
	miangedan/services/secretref v0.0.0 // indirect
)

replace miangedan/services/identity => ../identity
replace miangedan/services/notify => ../notify
replace miangedan/services/region => ../region
replace miangedan/services/secretref => ../secretref
