package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	//	"strings"
	"time"

	"github.com/astaxie/beego"
)

const (
	publicKey = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDTT02lsLXk3Z1csbtvrmiRKPN5
XSOq43TmlQdvTQ62y7113pLacbeb2/g912uN1l5XdgU6Yt8dCBFghzV0OqnaLSXW
I6gS4R35/1Ww5IwMPa7JIbRsjdcwghycTwp5Smae0doBiivYYMBVWDuAw4+pAU1L
t1e8djXNjllgvvSDRwIDAQAB
-----END PUBLIC KEY-----`
)

/*
	query body define
*/
type everBrightBankQueryUser struct {
	UserId string `json:"userId"`
}

type everBrightBankQuerySession struct {
	IsNewSession bool                    `json:"isNewSession"`
	SessionId    string                  `json:"sessionId"`
	User         everBrightBankQueryUser `json:"user"`
	ReqDeviceId  string                  `json:"reqDeviceId"`
	Timestamp    string                  `json:"timestamp"`
}

type everBrightBankQueryRequest struct {
	Type        string `json:"type"`
	InputSpeech string `json:"inputSpeech"`
}

type everBrightBankQueryBody struct {
	Version string                     `json:"version"`
	Session everBrightBankQuerySession `json:"session"`
	Request everBrightBankQueryRequest `json:"request"`
}

/*
	response body define
*/
type everBrightBankRespSession struct {
	ShouldEndSession bool `json:"shouldEndSession"`
}

type everBrightBankRespOuterSpeech struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type everBrightBankRespResponse struct {
	OutputSpeech everBrightBankRespOuterSpeech `json:"outputSpeech"`
}

type everBrightBankRespBody struct {
	Version  string                     `json:"version"`
	Session  everBrightBankRespSession  `json:"session"`
	Response everBrightBankRespResponse `json:"response"`
}

func Base64Encode(s []byte) string {
	return base64.URLEncoding.EncodeToString(s)
}

func Base64WithRsaSignWithSha1(bodyJson []byte, pubKey string) (string, error) {
	block, _ := pem.Decode([]byte(pubKey))
	if block == nil {
		beego.Debug("rsa encode public key error")
		return "", errors.New("rsa encode public key error")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		beego.Debug("ParsePKIXPublicKey err", err)
		return "", err
	}

	// 对请求body进行sha1运算，得到值hashA
	h := sha1.New()
	h.Write(bodyJson)
	hashA := h.Sum(nil)
	beego.Debug("hashA =", hashA)

	// hashA进行rsa加密，加密后得到值sigA
	sigA, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), hashA[:])
	if err != nil {
		beego.Debug("Error from signing: %s\n", err)
		return "", err
	}
	beego.Debug("sigA =", sigA)

	// 对sigA进行base64运算，得到sigA-1
	sigA1 := base64.StdEncoding.EncodeToString(sigA)
	beego.Debug("sigA1 =", sigA1)

	// 对sigA-1进行urlencode，得到sigB
	sigB := url.QueryEscape(sigA1)

	return sigB, nil
}

func EverBrightBank(query, agentId, sessionId string) (string, error) {
	beego.Debug("EverBrightBank robot called")
	beego.Debug("query is", query)

	EverBrightBankURL := "http://ccs.test.cebbank.com:9001/robot/index.do"

	// queryBody 的组装
	var queryBody everBrightBankQueryBody
	queryBody.Version = "1.0"
	queryBody.Session.IsNewSession = true
	queryBody.Session.SessionId = sessionId
	queryBody.Session.User.UserId = "1"
	queryBody.Session.ReqDeviceId = "2"
	queryBody.Session.Timestamp = string(strconv.FormatInt(time.Now().Unix(), 10))
	queryBody.Request.Type = "PlainText"
	queryBody.Request.InputSpeech = query

	bodyJson, err := json.Marshal(queryBody)
	if err != nil {
		beego.Debug("json marshal querybody err", err)
		return "", err
	}

	// body 加密
	sigB, err := Base64WithRsaSignWithSha1(bodyJson, publicKey)
	if err != nil {
		beego.Debug("query body encoding err", err)
		return "", err
	}
	beego.Debug("sigB = ", sigB)

	req, err := http.NewRequest("POST", EverBrightBankURL+"?signature="+sigB, bytes.NewBuffer(bodyJson))
	if err != nil {
		beego.Debug("http.NewRequest POST error: ", err)
		return "", err
	}

	// header设置
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Charset", "utf-8")
	req.Header.Set("Accept-Language", "zh-CN")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		beego.Debug("client.Do req error: ", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		beego.Debug("ioutil.ReadAll resp.Body error: ", err)
		return "", err
	}
	beego.Debug("response Body:", string(body))

	var respBody everBrightBankRespBody
	json.Unmarshal([]byte(body), &respBody)

	answer := respBody.Response.OutputSpeech.Text

	beego.Debug(respBody.Session.ShouldEndSession)
	beego.Debug(respBody.Response.OutputSpeech.Type)
	beego.Debug(respBody.Response.OutputSpeech.Text)

	return answer, nil
}

const (
	REGEX_ENGLISH_SENT = "^[[:alpha:][:punct:][:digit:]]+(\\s[[:alpha:][:punct:][:digit:]]+)*$"
	REGEX_JAPAN        = "[\u3040-\u309f\u30a0-\u30ff\u31f0-\u31ff\u3190-\u319f\uff00-\uffef]+"
)

func CheckLang(query string) {
	isJP, _ := regexp.MatchString(REGEX_JAPAN, query)
	if isJP {
		fmt.Println("japan")
		return
	}
	isEN, _ := regexp.MatchString(REGEX_ENGLISH_SENT, query)
	if isEN {
		fmt.Println("english")
		return
	}
	fmt.Println("chinese")
}

func patternSeg(t string) []string {
	//        begin := time.Now().UnixNano()
	t_lst := strings.Split(t, "")
	tmp := ""
	var temp_lst []string
	pre := ""
	for _, i := range t_lst {
		if i == "@" {
			if tmp != "" {
				temp_lst = append(temp_lst, tmp)
			}
			tmp = "@"
			pre = "@"
		} else if len(i) == 3 && pre == "@" {
			if tmp != "" {
				temp_lst = append(temp_lst, tmp)
			}
			tmp = i
			pre = ""
		} else {
			tmp += i
		}
	}
	if tmp != "" {
		temp_lst = append(temp_lst, tmp)
	}
	//      end := time.Now().UnixNano()
	//      beego.Debug("SSSSSSSKKKKKKKKKKKKKKK", temp_lst, end-begin)
	return temp_lst
}

func PatternSegment(template string) []string {
	template_lst := strings.Split(template, "[")
	var temp_lst []string
	//for _, t := range template_lst {
	i := 0
	t := template_lst[i]

	for true {

		//beego.Debug("MMMMMMMMMMMMMM", t)
		if strings.Contains(t, "]") {
			t = "[" + t
			index := strings.Index(t, "]")
			temp_lst = append(temp_lst, t[0:index+1])
			tt := t[index+1:]
			if tt != "" {
				//beego.Debug("YYYYYYYYYYY", tt)
				if !strings.Contains(tt, "<<") {
					//beego.Debug("LLLLLLLLLLLL")
					segs := patternSeg(tt)
					temp_lst = append(temp_lst, segs...)
				} else {
					//beego.Debug("JJJJJJJJJJJJJJJ")
					t = tt
					continue
				}
			}
		} else if strings.Contains(t, "<<") {
			lst := strings.Split(t, "<<")
			//beego.Debug("FFFFFFFFFFFFF", t, lst)
			for _, s := range lst {
				//beego.Debug("PPPPPPPPPPPP", s)
				if strings.Contains(s, ">>") {
					s = "<<" + s
					index := strings.Index(s, ">>")
					temp_lst = append(temp_lst, s[0:index+2])
					ss := s[index+2:]
					if ss != "" {
						//beego.Debug("HHHHHHHHHHHHH", ss)
						if !strings.Contains(ss, "<<") {
							//beego.Debug("TTTTTTTTTT")
							segs := patternSeg(ss)
							temp_lst = append(temp_lst, segs...)
						} else {
							//beego.Debug("RRRRRRRTTTTTTTTTT")
							t = ss
							continue
						}
					}
				} else if s != "" {
					segs := patternSeg(s)
					temp_lst = append(temp_lst, segs...)
				}
			}
		} else if t != "" {
			segs := patternSeg(t)
			temp_lst = append(temp_lst, segs...)
		}
		i += 1
		if i == len(template_lst) {
			break
		}
		t = template_lst[i]
	}

	//beego.Debug("WWWWWWWWWWWWWWWWWW", temp_lst)
	return temp_lst
}

func dealparam(template string) []string {
	//reg_param := regexp.MustCompile(`(@[\w\-\._]{1,30}[:?\w\-\._]{0,30})`)
	if !strings.Contains(template, "@") {
		return []string{template}
	}

	lst := strings.Split(template, "")
	for _, v := range lst {
		fmt.Println(v)
	}

	var ret_lst []string
	var pre, tmp string
	for _, l := range lst {
		if l == "@" {
			pre = "@"
			ret_lst = append(ret_lst, tmp)
			fmt.Println(tmp)
			tmp = ""
		} else if len(l) == 3 && pre == "@" {
			pre = ""
			tmp = l
		} else if pre == "@" {
			continue
		} else if len(l) == 3 {
			tmp += l
		}
	}

	if len(tmp) > 0 {
		ret_lst = append(ret_lst, tmp)
	}

	return ret_lst
}

type item struct {
	words []string
	value string
}

type items []item

func (s items) Len() int {
	return len(s)
}

func (s items) Less(i, j int) bool {
	index := 0
	if len(s[i].words) == len(s[j].words) {
		for ; index < len(s[i].words) && index < len(s[j].words); index++ {
			if len((s[i].words)[index]) == len((s[j].words)[index]) {
				continue
			}
			return len((s[i].words)[index]) > len((s[j].words)[index])
		}

		if index == len(s[i].words) && index == len(s[j].words) {
			if s[i].value == "" {
				return false
			} else if s[j].value == "" {
				return true
			}
			if strings.Contains(s[i].value, "@sys.date-time") {
				return true
			}
			if strings.Contains(s[j].value, "@sys.date-time") {
				return false
			}
			if strings.Contains(s[i].value, "@sys.") && !strings.Contains(s[i].value, "@sys.entity.") {
				return true
			}
			if strings.Contains(s[j].value, "@sys.") && !strings.Contains(s[j].value, "@sys.entity.") {
				return true
			}
			return len(s[i].value) > len(s[j].value)
		}

		if index == len(s[i].words) {
			return false
		}
	} else {
		if s[i].value == "" {
			s[i].words = []string{}
		}
		if s[j].value == "" {
			s[j].words = []string{}
		}
		return len(s[i].words) > len(s[j].words)
	}
	return true
}

func (s items) Swap(i, j int) {
	s[i].words, s[j].words = s[j].words, s[i].words
	s[i].value, s[j].value = s[j].value, s[i].value
}

func DealPattern(seg string) []string {
	ret_lst := strings.Split(seg, "|")
	var v_tmp_item []item
	for _, vt := range ret_lst {
		t_item := item{words: dealparam(vt), value: vt}
		v_tmp_item = append(v_tmp_item, t_item)
	}
	sort.Sort(items(v_tmp_item))
	ret_lst = []string{}
	for _, vti := range v_tmp_item {
		ret_lst = append(ret_lst, vti.value)
	}

	return ret_lst
}

type MyRegexp struct {
	*regexp.Regexp
}

//add a new method to our new regular expression type
func (r *MyRegexp) FindStringSubmatchMap(s string) map[string]string {
	captures := make(map[string]string)

	match := r.FindStringSubmatch(s)
	if match == nil {
		return captures
	}

	for i, name := range r.SubexpNames() {
		// Ignore the whole regexp match and unnamed groups
		if i == 0 || name == "" {
			continue
		}

		captures[name] = match[i]

	}
	return captures
}

func main() {
	if strings.HasPrefix("niha", "ni") {
		fmt.Println("left")
	}

	if strings.HasSuffix("niha", "ha") {
		fmt.Println("right")
	}

	//	tmp := "[出发|起飞|][目的|目地|][地|][是|到|去|飞]@sys.entity.city:DestCity[机票|的机票|的飞机票|飞机票]有没有"
	//	temp_lst := PatternSegment(tmp)
	//	for _, v := range temp_lst {
	//		fmt.Println(v)
	//	}

	//	for _, v := range DealPattern("机票|的机票|的飞机票|飞机票") {
	//		fmt.Println(v)
	//	}

	//	reg_param := regexp.MustCompile("(?:机票)")
	//	ps := reg_param.FindAllString("怎么去预定机票，今后还有机票吗？", -1)
	//	for _, p := range ps {
	//		fmt.Println(p)
	//	}

	//	reg := MyRegexp{reg_param}
	//	params := reg.FindStringSubmatchMap("怎么去预定机票，今后还有机票吗？")
	//	for k, v := range params {
	//		fmt.Println(k + "\t" + v)
	//	}
	//	EverBrightBank("最近有什么热销理财", "123", "123")
	//	data := []byte("光大银行，你好")
	//	context := "光大银行，你好f58be"
	//	context_str := strings.Split(context, "")
	//	fmt.Println(context_str)
	//	CheckLang("1")
	//	CheckLang("の")
}
