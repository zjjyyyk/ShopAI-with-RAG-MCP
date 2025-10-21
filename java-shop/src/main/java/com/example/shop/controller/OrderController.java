package com.example.shop.controller;

import com.example.shop.model.Order;
import com.example.shop.service.OrderService;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * 订单 API 控制器
 */
@RestController
@RequestMapping("/api/orders")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class OrderController {

    private final OrderService orderService;

    @PostMapping
    public ResponseEntity<?> createOrder(@RequestBody CreateOrderRequest request) {
        try {
            Order order = orderService.createOrder(
                request.getProductId(),
                request.getQuantity(),
                request.getCustomerName(),
                request.getCustomerPhone(),
                request.getShippingAddress()
            );
            return ResponseEntity.ok(order);
        } catch (Exception e) {
            Map<String, String> error = new HashMap<>();
            error.put("error", e.getMessage());
            return ResponseEntity.badRequest().body(error);
        }
    }

    @GetMapping
    public ResponseEntity<List<Order>> getAllOrders() {
        return ResponseEntity.ok(orderService.getAllOrders());
    }

    @GetMapping("/{orderNumber}")
    public ResponseEntity<?> getOrderByNumber(@PathVariable String orderNumber) {
        return orderService.getOrderByOrderNumber(orderNumber)
            .map(ResponseEntity::ok)
            .orElse(ResponseEntity.notFound().build());
    }

    @DeleteMapping("/{orderNumber}")
    public ResponseEntity<?> cancelOrder(@PathVariable String orderNumber) {
        try {
            Order order = orderService.cancelOrder(orderNumber);
            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("message", "订单已成功取消");
            response.put("order", order);
            return ResponseEntity.ok(response);
        } catch (Exception e) {
            Map<String, String> error = new HashMap<>();
            error.put("error", e.getMessage());
            return ResponseEntity.badRequest().body(error);
        }
    }

    @PutMapping("/{orderNumber}/status")
    public ResponseEntity<?> updateOrderStatus(
            @PathVariable String orderNumber,
            @RequestBody UpdateStatusRequest request) {
        try {
            Order order = orderService.updateOrderStatus(orderNumber, request.getStatus());
            return ResponseEntity.ok(order);
        } catch (Exception e) {
            Map<String, String> error = new HashMap<>();
            error.put("error", e.getMessage());
            return ResponseEntity.badRequest().body(error);
        }
    }

    @Data
    public static class CreateOrderRequest {
        private Long productId;
        private Integer quantity;
        private String customerName;
        private String customerPhone;
        private String shippingAddress;
    }

    @Data
    public static class UpdateStatusRequest {
        private Order.OrderStatus status;
    }
}
