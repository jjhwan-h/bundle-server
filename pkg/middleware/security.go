package middleware

import (
	"github.com/jjhwan-h/bundle-server/config"

	"github.com/gin-contrib/secure"
	"github.com/gin-gonic/gin"
)

func Security() gin.HandlerFunc {
	// https://pkg.go.dev/github.com/gin-contrib/secure#Config
	return secure.New(secure.Config{
		AllowedHosts:         config.Cfg.Security.AllowedHosts,         // 잘못된 HOST의 접근 필터링 => 올바른 보안대책필요
		SSLRedirect:          config.Cfg.Security.SSLRedirect,          // https request만 허용
		SSLHost:              config.Cfg.Security.SSLHost,              // SSLRedirect가 true일때, 리디렉션 대상 호스트를 명시적으로 지정
		STSSeconds:           int64(config.Cfg.Security.STSSeconds),    // HSTS헤더 유지시간
		STSIncludeSubdomains: config.Cfg.Security.STSIncludeSubdomains, // HSTS 서브도메인에 동일하게 적용
		FrameDeny:            config.Cfg.Security.FrameDeny,            // iframe 삽입 차단
		ContentTypeNosniff:   config.Cfg.Security.ContentTypeNoSniff,   // MIME type sniffing 방지
		// BrowserXssFilter:     true, // 최신 브라우저에서는 이 헤더 무시
		// ContentSecurityPolicy: "default-src 'self'", // 웹페이지가 로딩할 수 있는 콘텐츠 제한
		IENoOpen:        config.Cfg.Security.IENoOpen,        // IE가 다운로드한 파일을 자체적으로 실행하지 않도록 설정
		ReferrerPolicy:  config.Cfg.Security.ReferrerPolicy,  // 외부사이트로 이동할때 referrer헤더에 어떤 정보를 포함할지 결정
		SSLProxyHeaders: config.Cfg.Security.SSLProxyHeaders, // 리버스 프록시로부터의 헤더를 읽어 "https"판단
	})
}
