module miangedan/services/identity

go 1.26

require (
	miangedan/services/notify v0.0.0
	miangedan/services/region v0.0.0
	miangedan/services/secretref v0.0.0
)

replace miangedan/services/notify => ../notify
replace miangedan/services/region => ../region
replace miangedan/services/secretref => ../secretref
