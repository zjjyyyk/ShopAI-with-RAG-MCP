package com.example.shop.controller;

import com.example.shop.service.AiChatService;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

/**
 * AI 客服 API 控制器
 */
@RestController
@RequestMapping("/api/chat")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class ChatController {

    private final AiChatService aiChatService;
    
    @Value("${chat.max-history-rounds:20}")
    private int maxHistoryRounds;

    @PostMapping
    public ResponseEntity<Map<String, String>> chat(@RequestBody ChatRequest request) {
        String reply = aiChatService.sendMessage(
            request.getMessage(),
            request.getUserId() != null ? request.getUserId() : "anonymous",
            request.getSessionId(),
            request.getHistory()  // 传递历史消息
        );

        Map<String, String> response = new HashMap<>();
        response.put("reply", reply);
        return ResponseEntity.ok(response);
    }
    
    @GetMapping("/config")
    public ResponseEntity<Map<String, Object>> getConfig() {
        Map<String, Object> config = new HashMap<>();
        config.put("maxHistoryRounds", maxHistoryRounds);
        return ResponseEntity.ok(config);
    }

    @Data
    public static class ChatRequest {
        private String message;
        private String userId;
        private String sessionId;
        private java.util.List<HistoryMessage> history;  // 添加历史消息字段
    }
    
    @Data
    public static class HistoryMessage {
        private String role;
        private String content;
    }
}
