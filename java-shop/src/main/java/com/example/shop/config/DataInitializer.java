package com.example.shop.config;

import com.example.shop.model.Order;
import com.example.shop.model.Product;
import com.example.shop.repository.OrderRepository;
import com.example.shop.repository.ProductRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.CommandLineRunner;
import org.springframework.stereotype.Component;

import java.math.BigDecimal;
import java.time.LocalDateTime;

/**
 * 数据初始化 - 添加示例商品和订单
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class DataInitializer implements CommandLineRunner {

    private final ProductRepository productRepository;
    private final OrderRepository orderRepository;

    @Override
    public void run(String... args) {
        if (productRepository.count() == 0) {
            log.info("初始化商品数据...");
            initProducts();
        }
        
        if (orderRepository.count() == 0) {
            log.info("初始化示例订单...");
            initOrders();
        }
    }

    private void initProducts() {
        // 自行车
        Product bike1 = new Product();
        bike1.setName("山地自行车 Pro X1");
        bike1.setDescription("专业级山地自行车,适合越野和山地骑行。配备27速变速系统,前后避震,碳纤维车架。");
        bike1.setPrice(new BigDecimal("3999.00"));
        bike1.setStock(50);
        bike1.setCategory("自行车");
        bike1.setImageUrl("/images/bike1.jpg");
        bike1.setSpecifications("车架材质:碳纤维 | 轮径:27.5英寸 | 变速:27速 | 重量:12kg");
        productRepository.save(bike1);

        Product bike2 = new Product();
        bike2.setName("公路自行车 Speed R3");
        bike2.setDescription("高性能公路自行车,轻量化设计,适合长途骑行和竞速。18速变速系统。");
        bike2.setPrice(new BigDecimal("2899.00"));
        bike2.setStock(30);
        bike2.setCategory("自行车");
        bike2.setImageUrl("/images/bike2.jpg");
        bike2.setSpecifications("车架材质:铝合金 | 轮径:700C | 变速:18速 | 重量:9kg");
        productRepository.save(bike2);

        Product bike3 = new Product();
        bike3.setName("城市通勤自行车 City Easy");
        bike3.setDescription("舒适的城市通勤自行车,配备货架和挡泥板,适合日常代步。");
        bike3.setPrice(new BigDecimal("1299.00"));
        bike3.setStock(80);
        bike3.setCategory("自行车");
        bike3.setImageUrl("/images/bike3.jpg");
        bike3.setSpecifications("车架材质:钢制 | 轮径:26英寸 | 变速:单速 | 重量:15kg");
        productRepository.save(bike3);

        // 头盔
        Product helmet1 = new Product();
        helmet1.setName("专业骑行头盔 SafeRide Pro");
        helmet1.setDescription("轻量化设计,通风良好,符合国际安全标准。内置LED尾灯,提高夜间骑行安全性。");
        helmet1.setPrice(new BigDecimal("299.00"));
        helmet1.setStock(100);
        helmet1.setCategory("配件");
        helmet1.setImageUrl("/images/helmet1.jpg");
        helmet1.setSpecifications("尺寸:可调节(54-62cm) | 重量:280g | 认证:CE/CPSC");
        productRepository.save(helmet1);

        // 配件
        Product accessory1 = new Product();
        accessory1.setName("自行车锁 SecureLock Max");
        accessory1.setDescription("高强度U型锁,防盗级别高,配备钥匙和密码双重保护。");
        accessory1.setPrice(new BigDecimal("199.00"));
        accessory1.setStock(150);
        accessory1.setCategory("配件");
        accessory1.setImageUrl("/images/lock1.jpg");
        accessory1.setSpecifications("材质:高碳钢 | 锁径:13mm | 长度:230mm | 防盗等级:10级");
        productRepository.save(accessory1);

        log.info("商品数据初始化完成,共 {} 件商品", productRepository.count());
    }

    private void initOrders() {
        // 创建几个示例订单
        Product bike = productRepository.findById(1L).orElse(null);
        if (bike != null) {
            Order order1 = new Order();
            order1.setOrderNumber("ORD-001");
            order1.setProduct(bike);
            order1.setQuantity(1);
            order1.setTotalPrice(bike.getPrice());
            order1.setCustomerName("张三");
            order1.setCustomerPhone("13800138000");
            order1.setShippingAddress("北京市朝阳区XX路XX号");
            order1.setStatus(Order.OrderStatus.SHIPPED);
            order1.setCreatedAt(LocalDateTime.now().minusDays(2));
            order1.setUpdatedAt(LocalDateTime.now().minusDays(1));
            orderRepository.save(order1);

            Order order2 = new Order();
            order2.setOrderNumber("ORD-002");
            order2.setProduct(bike);
            order2.setQuantity(1);
            order2.setTotalPrice(bike.getPrice());
            order2.setCustomerName("李四");
            order2.setCustomerPhone("13900139000");
            order2.setShippingAddress("上海市浦东新区YY路YY号");
            order2.setStatus(Order.OrderStatus.PENDING);
            order2.setCreatedAt(LocalDateTime.now().minusHours(5));
            order2.setUpdatedAt(LocalDateTime.now().minusHours(5));
            orderRepository.save(order2);
        }

        log.info("示例订单初始化完成,共 {} 个订单", orderRepository.count());
    }
}
