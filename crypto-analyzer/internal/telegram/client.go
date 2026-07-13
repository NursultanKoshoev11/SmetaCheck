package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	token      string
	chatID     string
	baseURL    string
	httpClient *http.Client
}

type Update struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		MessageID int64  `json:"message_id"`
		Text      string `json:"text"`
		Chat      struct { ID int64 `json:"id"` } `json:"chat"`
	} `json:"message"`
}

type apiResponse[T any] struct { OK bool `json:"ok"`; Result T `json:"result"`; ErrorCode int `json:"error_code"`; Description string `json:"description"` }

func New(token, chatID string, timeout time.Duration) *Client { return &Client{token:strings.TrimSpace(token),chatID:strings.TrimSpace(chatID),baseURL:"https://api.telegram.org",httpClient:&http.Client{Timeout:timeout+40*time.Second}} }
func (c *Client) Enabled() bool { return c.token!=""&&c.chatID!="" }
func (c *Client) ChatID() string{return c.chatID}
func (c *Client) Send(ctx context.Context,text string)(int,error){if !c.Enabled(){return 0,fmt.Errorf("Telegram is not configured")};chunks:=splitMessage(text,3900);lastCode:=0;for _,chunk:=range chunks{payload:=map[string]any{"chat_id":c.chatID,"text":chunk,"disable_web_page_preview":true};body,_:=json.Marshal(payload);req,err:=http.NewRequestWithContext(ctx,http.MethodPost,c.endpoint("sendMessage"),bytes.NewReader(body));if err!=nil{return lastCode,err};req.Header.Set("Content-Type","application/json");resp,err:=c.httpClient.Do(req);if err!=nil{return lastCode,err};lastCode=resp.StatusCode;responseBody,readErr:=io.ReadAll(io.LimitReader(resp.Body,1<<20));resp.Body.Close();if readErr!=nil{return lastCode,readErr};var result apiResponse[json.RawMessage];if err:=json.Unmarshal(responseBody,&result);err!=nil{return lastCode,fmt.Errorf("decode Telegram response: %w",err)};if resp.StatusCode<200||resp.StatusCode>=300||!result.OK{return lastCode,fmt.Errorf("Telegram API error %d: %s",result.ErrorCode,result.Description)}};return lastCode,nil}
func (c *Client) GetUpdates(ctx context.Context,offset int64)([]Update,error){if !c.Enabled(){return nil,nil};query:=url.Values{};query.Set("offset",strconv.FormatInt(offset,10));query.Set("timeout","30");query.Set("limit","50");query.Set("allowed_updates",`["message"]`);req,err:=http.NewRequestWithContext(ctx,http.MethodGet,c.endpoint("getUpdates")+"?"+query.Encode(),nil);if err!=nil{return nil,err};resp,err:=c.httpClient.Do(req);if err!=nil{return nil,err};defer resp.Body.Close();body,err:=io.ReadAll(io.LimitReader(resp.Body,2<<20));if err!=nil{return nil,err};var result apiResponse[[]Update];if err:=json.Unmarshal(body,&result);err!=nil{return nil,err};if resp.StatusCode<200||resp.StatusCode>=300||!result.OK{return nil,fmt.Errorf("Telegram getUpdates error %d: %s",result.ErrorCode,result.Description)};return result.Result,nil}
func (c *Client) endpoint(method string)string{return fmt.Sprintf("%s/bot%s/%s",c.baseURL,c.token,method)}
func splitMessage(text string,maxRunes int)[]string{if maxRunes<=0{maxRunes=3900};runes:=[]rune(text);if len(runes)<=maxRunes{return []string{text}};result:=make([]string,0,len(runes)/maxRunes+1);for len(runes)>0{end:=maxRunes;if end>len(runes){end=len(runes)};if end<len(runes){for i:=end;i>end-400&&i>0;i--{if runes[i-1]=='\n'{end=i;break}}};result=append(result,string(runes[:end]));runes=runes[end:]};return result}
