package com.example.shop.service;

import com.example.shop.model.Product;
import com.example.shop.repository.ProductRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Optional;

/**
 * 商品服务
 */
@Service
@RequiredArgsConstructor
@Slf4j
public class ProductService {

    private final ProductRepository productRepository;

    /**
     * 获取所有商品
     */
    public List<Product> getAllProducts() {
        return productRepository.findAll();
    }

    /**
     * 根据ID获取商品
     */
    public Optional<Product> getProductById(Long id) {
        return productRepository.findById(id);
    }

    /**
     * 根据类别获取商品
     */
    public List<Product> getProductsByCategory(String category) {
        return productRepository.findByCategory(category);
    }

    /**
     * 搜索商品
     */
    public List<Product> searchProducts(String keyword) {
        return productRepository.findByNameContainingIgnoreCase(keyword);
    }

    /**
     * 创建商品
     */
    @Transactional
    public Product createProduct(Product product) {
        log.info("创建商品: {}", product.getName());
        return productRepository.save(product);
    }

    /**
     * 更新库存
     */
    @Transactional
    public void updateStock(Long productId, int quantity) {
        Product product = productRepository.findById(productId)
            .orElseThrow(() -> new RuntimeException("商品不存在"));
        
        int newStock = product.getStock() + quantity;
        if (newStock < 0) {
            throw new RuntimeException("库存不足");
        }
        
        product.setStock(newStock);
        productRepository.save(product);
        log.info("更新商品库存: {} - {}", product.getName(), newStock);
    }
}
