package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "encoding/base64"
    "net/smtp"
    "io"
)

type RequestBody struct {
    Account            string `json:"account"`
    Pwd                string `json:"pwd"`
    AgreementAgreement int    `json:"agreement_agreement"`
    IsLocal            bool   `json:"is_local"`
}

type ResponseData struct {
    Token string `json:"token"`
	UID string `json:"uid"`
	Timestamp int64 `json:"timestamp"`
}

type ResponseBody struct {
    HasError bool         `json:"hasError"`
    Code     int          `json:"code"`
    Msg      string       `json:"msg"`
    Data     ResponseData `json:"data"`
}

type TokenPayload struct {
    UID       string `json:"uid"`
    Timestamp int64  `json:"timestamp"`
    Token     string `json:"token"`
    Client    string `json:"client"`
    Version   string `json:"version"`
    Language  string `json:"language"`
}

func main() {
    pwd := os.Getenv("SEMS_PWD")
    if pwd == "" {
        fmt.Println("SEMS_PWD environment variable not set")
        return
    }

    reqBody := RequestBody{
        Account:            "rcrukshan17@gmail.com",
        Pwd:                pwd,
        AgreementAgreement: 0,
        IsLocal:            false,
    }

    jsonData, err := json.Marshal(reqBody)
    if err != nil {
        fmt.Println("Error marshaling request body:", err)
        return
    }

    req, err := http.NewRequest("POST", "https://hk.semsportal.com/api/v2/common/crosslogin", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Println("Error creating request:", err)
        return
    }

    tokenValue := "eyJ1aWQiOiIiLCJ0aW1lc3RhbXAiOjAsInRva2VuIjoiIiwiY2xpZW50Ijoid2ViIiwidmVyc2lvbiI6IiIsImxhbmd1YWdlIjoiZW4ifQ=="
    req.Header.Set("accept", "application/json")
    req.Header.Set("content-type", "application/json")
    req.Header.Set("token", tokenValue)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("Error making request:", err)
        return
    }
    defer resp.Body.Close()

    var respBody ResponseBody
    if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
        fmt.Println("Error decoding response:", err)
        return
    }

	token := respBody.Data.Token
	uid := respBody.Data.UID
	timestamp := respBody.Data.Timestamp

	payload := TokenPayload{
        UID:       uid,
        Timestamp: timestamp,
        Token:     token,
        Client:    "web",
        Version:   "",
        Language:  "en",
    }

    payloadJSON, err := json.Marshal(payload)
    if err != nil {
        fmt.Println("Error marshaling payload:", err)
        return
    }

    encoded := base64.StdEncoding.EncodeToString(payloadJSON)

    // --- Powerflow API call ---
    // Prepare form data
    form := []byte("PowerStationId=556432f6-84fe-4bd9-b52e-e1b005a8e2d9")

    req2, err := http.NewRequest("POST", "https://hk.semsportal.com/api/v2/PowerStation/GetPowerflow", bytes.NewBuffer(form))
    if err != nil {
        fmt.Println("Error creating Powerflow request:", err)
        return
    }
    req2.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
    req2.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
    req2.Header.Set("token", encoded)

    resp2, err := client.Do(req2)
    if err != nil {
        fmt.Println("Error making Powerflow request:", err)
        return
    }
    defer resp2.Body.Close()

    // Read the full response body
    powerflowRespBody, err := io.ReadAll(resp2.Body)
    if err != nil {
        fmt.Println("Error reading Powerflow response body:", err)
        return
    }

    // Parse for gridStatus and grid
    var pfResp struct {
        Data struct {
            Powerflow struct {
                Grid       string `json:"grid"`
                GridStatus int    `json:"gridStatus"`
            } `json:"powerflow"`
        } `json:"data"`
    }
    if err := json.Unmarshal(powerflowRespBody, &pfResp); err != nil {
        fmt.Println("Error decoding Powerflow response:", err)
        return
    }

    fmt.Println("gridStatus:", pfResp.Data.Powerflow.GridStatus)
    fmt.Println("grid:", pfResp.Data.Powerflow.Grid)

    // Send email if gridStatus != 1
    if pfResp.Data.Powerflow.GridStatus != 1 {
        smtpHost := os.Getenv("SMTP_HOST")
        smtpPort := os.Getenv("SMTP_PORT")
        smtpUser := os.Getenv("SMTP_USER")
        smtpPass := os.Getenv("SMTP_PASS")
        mailTo := os.Getenv("ALERT_EMAIL_TO")
        mailFrom := os.Getenv("ALERT_EMAIL_FROM")
        if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" || mailTo == "" || mailFrom == "" {
            fmt.Println("SMTP or email environment variables not set")
            return
        }

        subject := "SEMS Alert: gridStatus is not 1"
        body := string(powerflowRespBody)
        msg := "From: " + mailFrom + "\r\n" +
            "To: " + mailTo + "\r\n" +
            "Subject: " + subject + "\r\n" +
            "MIME-Version: 1.0\r\n" +
            "Content-Type: text/plain; charset=\"utf-8\"\r\n" +
            "\r\n" +
            body + "\r\n"

        addr := smtpHost + ":" + smtpPort
        auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

        err = smtp.SendMail(addr, auth, mailFrom, []string{mailTo}, []byte(msg))
        if err != nil {
            fmt.Println("Error sending alert email:", err)
        } else {
            fmt.Println("Alert email sent.")
        }
    }
}