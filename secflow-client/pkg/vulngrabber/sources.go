package vulngrabber

// sources.go registers all vulnerability data sources.
// Only go-rod based crawlers are registered for WAF bypass capability.
// Standard HTTP client crawlers have been removed.

func init() {
	// ── International Sources (Rod-based) ─────────────────────────────────────
	Register("avd-rod", func() Grabber { return NewAVDCrawlerRod() })
	Register("seebug-rod", func() Grabber { return NewSeebugCrawlerRod() })
	Register("ti-rod", func() Grabber { return NewTiCrawlerRod() })
	Register("nox-rod", func() Grabber { return NewTiCrawlerRod() })
	Register("kev-rod", func() Grabber { return NewKEVCrawlerRod() })
	Register("struts2-rod", func() Grabber { return NewStruts2CrawlerRod() })
	Register("chaitin-rod", func() Grabber { return NewChaitinCrawlerRod() })
	Register("oscs-rod", func() Grabber { return NewOSCSCrawlerRod() })
	Register("threatbook-rod", func() Grabber { return NewThreatBookCrawlerRod() })
	Register("venustech-rod", func() Grabber { return NewVenustechCrawlerRod() })

	// ── Chinese Security Vendors (Rod-based) ─────────────────────────────────
	Register("cnvd-rod", func() Grabber { return NewCNVDCrawlerRod() })
	Register("cnnvd-rod", func() Grabber { return NewCNNVDCrawlerRod() })
	Register("nsfocus-rod", func() Grabber { return NewNsfocusCrawlerRod() })
	Register("qianxin-rod", func() Grabber { return NewQianxinCrawlerRod() })
	Register("antiy-rod", func() Grabber { return NewAntiyCrawlerRod() })
	Register("dbappsecurity-rod", func() Grabber { return NewDbappsecurityCrawlerRod() })

	// ── NEW: Additional International Sources (Rod-based) ────────────────────
	Register("nvd-rod", func() Grabber { return NewNVDCrawlerRod() })
	Register("vulhub-rod", func() Grabber { return NewVulHubCrawlerRod() })
	Register("packetstorm-rod", func() Grabber { return NewPacketStormCrawlerRod() })
	Register("exploitdb-rod", func() Grabber { return NewExploitDBCrawlerRod() })

	// NOTE: qianxin-weekly-rod has been moved to articlegrabber package
	// NOTE: 0day-rod and 360cert-rod have been removed
}
