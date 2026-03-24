package vulngrabber

import (
	"time"
	"unicode"
)

// ── OSCS Types ────────────────────────────────────────────────────────────────

type oscsListResp struct {
	Data struct {
		Total int `json:"total"`
		Data  []*struct {
			Project          []interface{} `json:"project"`
			Id               string        `json:"id"`
			Title            string        `json:"title"`
			Url              string        `json:"url"`
			Mps              string        `json:"mps"`
			IntelligenceType int           `json:"intelligence_type"`
			PublicTime       time.Time     `json:"public_time"`
			IsPush           int           `json:"is_push"`
			IsPoc            int           `json:"is_poc"`
			IsExp            int           `json:"is_exp"`
			Level            string        `json:"level"`
			CreatedAt        time.Time     `json:"created_at"`
			UpdatedAt        time.Time     `json:"updated_at"`
			IsSubscribe      int           `json:"is_subscribe"`
		} `json:"data"`
	} `json:"data"`
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Time    int    `json:"time"`
	Info    string `json:"info"`
}

type oscsDetailResp struct {
	Data []*struct {
		AttackVector           string        `json:"attack_vector"`
		CvssVector             string        `json:"cvss_vector"`
		Exp                    bool          `json:"exp"`
		ExploitRequirementCost string        `json:"exploit_requirement_cost"`
		Exploitability         string        `json:"exploitability"`
		ScopeInfluence         string        `json:"scope_influence"`
		Source                 string        `json:"source"`
		VulnType               string        `json:"vuln_type"`
		CvssScore              float64       `json:"cvss_score"`
		CveId                  string        `json:"cve_id"`
		VulnCveId              string        `json:"vuln_cve_id"`
		CnvdId                 string        `json:"cnvd_id"`
		IsOrigin               bool          `json:"is_origin"`
		Languages              []interface{} `json:"languages"`
		Description            string        `json:"description"`
		Effect                 []struct {
			AffectedAllVersion bool          `json:"affected_all_version"`
			AffectedVersion    string        `json:"affected_version"`
			EffectId           int           `json:"effect_id"`
			JavaQnList         []interface{} `json:"java_qn_list"`
			MinFixedVersion    string        `json:"min_fixed_version"`
			Name               string        `json:"name"`
			Solutions          []struct {
				Compatibility int    `json:"compatibility"`
				Description   string `json:"description"`
				Type          string `json:"type"`
			} `json:"solutions"`
		} `json:"effect"`
		Influence   int    `json:"influence"`
		Level       string `json:"level"`
		Patch       string `json:"patch"`
		Poc         bool   `json:"poc"`
		PublishTime int64  `json:"publish_time"`
		References  []struct {
			Name string `json:"name"`
			Url  string `json:"url"`
		} `json:"references"`
		SuggestLevel    string        `json:"suggest_level"`
		VulnSuggest     string        `json:"vuln_suggest"`
		Title           string        `json:"title"`
		Troubleshooting []string      `json:"troubleshooting"`
		VulnTitle       string        `json:"vuln_title"`
		VulnCodeUsage   []interface{} `json:"vuln_code_usage"`
		VulnNo          string        `json:"vuln_no"`
		SoulutionData   []string      `json:"soulution_data"`
	} `json:"data"`
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Time    int    `json:"time"`
	Info    string `json:"info"`
}

// ── Chaitin Types ─────────────────────────────────────────────────────────────

type ChaitinResp struct {
	Msg  string `json:"msg"`
	Data struct {
		Count    int         `json:"count"`
		Next     string      `json:"next"`
		Previous interface{} `json:"previous"`
		List     []struct {
			Id             string    `json:"id"`
			Title          string    `json:"title"`
			Summary        string    `json:"summary"`
			Severity       string    `json:"severity"`
			CtId           string    `json:"ct_id"`
			CveId          *string   `json:"cve_id"`
			References     *string   `json:"references"`
			DisclosureDate *string   `json:"disclosure_date"`
			CreatedAt      time.Time `json:"created_at"`
			UpdatedAt      time.Time `json:"updated_at"`
		} `json:"list"`
	} `json:"data"`
	Code int `json:"code"`
}

func ContainsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// ── ThreatBook Types ──────────────────────────────────────────────────────────

type threatBookHomepage struct {
	Data struct {
		HighRisk []struct {
			Id              string   `json:"id"`
			VulnNameZh      string   `json:"vuln_name_zh"`
			VulnUpdateTime  string   `json:"vuln_update_time"`
			Affects         []string `json:"affects"`
			VulnPublishTime string   `json:"vuln_publish_time,omitempty"`
			PocExist        bool     `json:"pocExist"`
			Solution        bool     `json:"solution"`
			Premium         bool     `json:"premium"`
			RiskLevel       string   `json:"riskLevel"`
			Is0Day          bool     `json:"is0day,omitempty"`
		} `json:"highrisk"`
	} `json:"data"`
	ResponseCode int `json:"response_code"`
}

// ── Ti (Qianxin) Types ────────────────────────────────────────────────────────

type tiVulnDetail struct {
	Id                int     `json:"id"`
	VulnName          string  `json:"vuln_name"`
	VulnNameEn        string  `json:"vuln_name_en"`
	QvdCode           string  `json:"qvd_code"`
	CveCode           string  `json:"cve_code"`
	CnvdId            *string `json:"cnvd_id"`
	CnnvdId           string  `json:"cnnvd_id"`
	ThreatCategory    string  `json:"threat_category"`
	TechnicalCategory string  `json:"technical_category"`
	ResidenceId       *int    `json:"residence_id"`
	RatingId          *int    `json:"rating_id"`
	NotShow           int     `json:"not_show"`
	PublishTime       string  `json:"publish_time"`
	Description       string  `json:"description"`
	DescriptionEn     string  `json:"description_en"`
	ChangeImpact      int     `json:"change_impact"`
	OperatorHid       string  `json:"operator_hid"`
	CreateHid         *string `json:"create_hid"`
	Channel           *string `json:"channel"`
	TrackingId        *string `json:"tracking_id"`
	Temp              int     `json:"temp"`
	OtherRating       int     `json:"other_rating"`
	CreateTime        string  `json:"create_time"`
	UpdateTime        string  `json:"update_time"`
	LatestUpdateTime  string  `json:"latest_update_time"`
	RatingLevel       string  `json:"rating_level"`
	VulnType          string  `json:"vuln_type"`
	PocFlag           int     `json:"poc_flag"`
	PatchFlag         int     `json:"patch_flag"`
	DetailFlag        int     `json:"detail_flag"`
	Tag               []struct {
		Name      string `json:"name"`
		FontColor string `json:"font_color"`
		BackColor string `json:"back_color"`
	} `json:"tag"`
	TagLen        int `json:"tag_len"`
	IsRatingLevel int `json:"is_rating_level"`
}

type tiOneDayResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    struct {
		VulnAddCount    int            `json:"vuln_add_count"`
		VulnUpdateCount int            `json:"vuln_update_count"`
		KeyVulnAddCount int            `json:"key_vuln_add_count"`
		PocExpAddCount  int            `json:"poc_exp_add_count"`
		PatchAddCount   int            `json:"patch_add_count"`
		KeyVulnAdd      []tiVulnDetail `json:"key_vuln_add"`
		PocExpAdd       []tiVulnDetail `json:"poc_exp_add"`
	} `json:"data"`
}