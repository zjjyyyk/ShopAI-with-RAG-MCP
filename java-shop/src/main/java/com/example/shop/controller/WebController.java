package com.example.shop.controller;

import com.example.shop.service.OrderService;
import com.example.shop.service.ProductService;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Controller;
import org.springframework.ui.Model;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;

/**
 * Web 页面控制器
 */
@Controller
@RequiredArgsConstructor
public class WebController {

    private final ProductService productService;
    private final OrderService orderService;

    @GetMapping("/")
    public String index(Model model) {
        model.addAttribute("products", productService.getAllProducts());
        return "index";
    }

    @GetMapping("/product/{id}")
    public String productDetail(@PathVariable Long id, Model model) {
        return productService.getProductById(id)
            .map(product -> {
                model.addAttribute("product", product);
                return "product-detail";
            })
            .orElse("redirect:/");
    }

    @GetMapping("/orders")
    public String orders(Model model) {
        model.addAttribute("orders", orderService.getAllOrders());
        return "orders";
    }
}
