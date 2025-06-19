package middleware

import (
	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
)

func Security() gin.HandlerFunc {
	// https://pkg.go.dev/github.com/gin-contrib/secure#Config
	return secure.New(secure.Config{
		AllowedHosts: []string{"127.0.0.1:4001", "localhost:4001"},
		//SSLRedirect:           true, // https request만 허용
		//SSLHost:               "ssl.example.com", // SSLRedirect가 true일때, 리디렉션 대상 호스트를 명시적으로 지정
		STSSeconds:           86400, // HSTS헤더 유지시간
		STSIncludeSubdomains: true,  // HSTS 서브도메인에 동일하게 적용
		FrameDeny:            true,  // iframe 삽입 차단
		ContentTypeNosniff:   true,  // MIME type sniffing 방지
		// BrowserXssFilter:     true, // 최신 브라우저에서는 이 헤더 무시
		// ContentSecurityPolicy: "default-src 'self'", // 웹페이지가 로딩할 수 있는 콘텐츠 제한
		IENoOpen:        true,                                            // IE가 다운로드한 파일을 자체적으로 실행하지 않도록 설정
		ReferrerPolicy:  "strict-origin-when-cross-origin",               // 외부사이트로 이동할때 referrer헤더에 어떤 정보를 포함할지 결정
		SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"}, // 리버스 프록시로부터의 헤더를 읽어 "https"판단
	})
}
