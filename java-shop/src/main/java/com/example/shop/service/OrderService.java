package com.example.shop.service;

import com.example.shop.model.Order;
import com.example.shop.model.Product;
import com.example.shop.repository.OrderRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;

/**
 * 订单服务
 */
@Service
@RequiredArgsConstructor
@Slf4j
public class OrderService {

    private final OrderRepository orderRepository;
    private final ProductService productService;

    /**
     * 创建订单
     */
    @Transactional
    public Order createOrder(Long productId, Integer quantity, String customerName, 
                           String customerPhone, String shippingAddress) {
        // 获取商品
        Product product = productService.getProductById(productId)
            .orElseThrow(() -> new RuntimeException("商品不存在"));

        // 检查库存
        if (product.getStock() < quantity) {
            throw new RuntimeException("库存不足");
        }

        // 计算总价
        BigDecimal totalPrice = product.getPrice().multiply(BigDecimal.valueOf(quantity));

        // 创建订单
        Order order = new Order();
        order.setProduct(product);
        order.setQuantity(quantity);
        order.setTotalPrice(totalPrice);
        order.setCustomerName(customerName);
        order.setCustomerPhone(customerPhone);
        order.setShippingAddress(shippingAddress);
        order.setStatus(Order.OrderStatus.PENDING);

        Order savedOrder = orderRepository.save(order);

        // 扣减库存
        productService.updateStock(productId, -quantity);

        log.info("创建订单成功: {}, 商品: {}, 数量: {}", 
                savedOrder.getOrderNumber(), product.getName(), quantity);

        return savedOrder;
    }

    /**
     * 获取所有订单
     */
    public List<Order> getAllOrders() {
        return orderRepository.findAll();
    }

    /**
     * 根据订单号查询订单
     */
    public Optional<Order> getOrderByOrderNumber(String orderNumber) {
        return orderRepository.findByOrderNumber(orderNumber);
    }

    /**
     * 更新订单状态
     */
    @Transactional
    public Order updateOrderStatus(String orderNumber, Order.OrderStatus newStatus) {
        Order order = orderRepository.findByOrderNumber(orderNumber)
            .orElseThrow(() -> new RuntimeException("订单不存在"));

        order.setStatus(newStatus);
        Order updatedOrder = orderRepository.save(order);

        log.info("更新订单状态: {} -> {}", orderNumber, newStatus);
        return updatedOrder;
    }

    /**
     * 取消订单
     */
    @Transactional
    public Order cancelOrder(String orderNumber) {
        Order order = orderRepository.findByOrderNumber(orderNumber)
            .orElseThrow(() -> new RuntimeException("订单不存在"));

        // 只有待处理和已确认的订单可以取消
        if (order.getStatus() != Order.OrderStatus.PENDING && 
            order.getStatus() != Order.OrderStatus.CONFIRMED) {
            throw new RuntimeException("订单状态不允许取消");
        }

        // 恢复库存
        productService.updateStock(order.getProduct().getId(), order.getQuantity());

        // 更新订单状态
        order.setStatus(Order.OrderStatus.CANCELLED);
        Order cancelledOrder = orderRepository.save(order);

        log.info("取消订单成功: {}", orderNumber);
        return cancelledOrder;
    }
}
