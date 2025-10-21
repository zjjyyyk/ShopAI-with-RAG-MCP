package com.example.shop.service;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.*;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.util.HashMap;
import java.util.Map;

/**
 * AI 客服服务 - 转发用户消息到 Go AI 服务
 */
@Service
@RequiredArgsConstructor
@Slf4j
public class AiChatService {

    @Value("${app.ai-service.url}")
    private String aiServiceUrl;

    private final RestTemplate restTemplate = new RestTemplate();

    /**
     * 发送消息到 AI 客服
     */
    public String sendMessage(String message, String userId, String sessionId) {
        return sendMessage(message, userId, sessionId, null);
    }

    /**
     * 发送消息到 AI 客服（带历史记录）
     */
    public String sendMessage(String message, String userId, String sessionId, java.util.List<?> history) {
        try {
            String url = aiServiceUrl + "/chat";

            // 构建请求体
            Map<String, Object> request = new HashMap<>();
            request.put("message", message);
            request.put("userId", userId);
            request.put("sessionId", sessionId != null ? sessionId : generateSessionId());
            if (history != null && !history.isEmpty()) {
                request.put("history", history);
            }

            // 设置请求头
            HttpHeaders headers = new HttpHeaders();
            headers.setContentType(MediaType.APPLICATION_JSON);

            HttpEntity<Map<String, Object>> entity = new HttpEntity<>(request, headers);

            log.info("转发消息到AI服务: {} - {}", userId, message);

            // 发送请求
            ResponseEntity<Map> response = restTemplate.exchange(
                url, 
                HttpMethod.POST, 
                entity, 
                Map.class
            );

            if (response.getStatusCode() == HttpStatus.OK && response.getBody() != null) {
                String reply = (String) response.getBody().get("reply");
                log.info("收到AI回复: {}", reply);
                return reply;
            } else {
                log.error("AI服务返回错误: {}", response.getStatusCode());
                return "抱歉,客服系统暂时不可用,请稍后再试。";
            }

        } catch (Exception e) {
            log.error("调用AI服务失败", e);
            return "抱歉,客服系统遇到问题,请稍后再试。";
        }
    }

    private String generateSessionId() {
        return "session-" + System.currentTimeMillis();
    }
}
