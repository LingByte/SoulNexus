// 心理健康助手机器人 - 温暖治愈的AI伙伴
// 专注于心理健康支持和情感陪伴
// 设计理念：温暖、治愈、专业、可爱，有手有脚可以移动

(function() {
    'use strict';
    
    // ==================== 配置 ====================
    const CONFIG = {
        botName: '小暖',
        botSize: 160,
        moveSpeed: 1.5,
        idleDialogInterval: 20000,
        autoMoveInterval: 15000,
        dialogDuration: 5000,
        walkingSpeed: 0.8
    };
    
    // ==================== 状态管理 ====================
    let state = {
        x: window.innerWidth - 220,
        y: window.innerHeight - 220,
        velocityX: 0,
        velocityY: 0,
        currentEmotion: 'calm',
        currentAction: 'idle',
        isMoving: false,
        isDragging: false,
        facingRight: true,
        dialogVisible: false,
        currentDialog: '',
        sessionStartTime: Date.now(),
        interactionCount: 0,
        lastInteractionTime: Date.now(),
        userMood: 'neutral',
        isWalking: false,
        targetX: 0,
        targetY: 0
    };
    
    // ==================== 对话内容库 ====================
    const DIALOGS = {
        greeting: [
            '你好！我是小暖，你的心理健康小助手 💙',
            '很高兴见到你！今天过得怎么样？😊',
            '嗨！我在这里陪伴你，有什么想聊的吗？',
            '你好呀！我是来给你带来温暖的小暖 🌟'
        ],
        
        supportive: [
            '记住，你很重要，你的感受也很重要 💝',
            '每一天都是新的开始，你做得很棒！',
            '深呼吸，一切都会好起来的 🌸',
            '你不是一个人，我会一直陪着你',
            '给自己一些时间，慢慢来就好 🕊️',
            '你的努力我都看得见，继续加油！'
        ],
        
        relaxation: [
            '要不要试试深呼吸？跟我一起... 吸气... 呼气... 🌊',
            '闭上眼睛，想象一个让你感到平静的地方',
            '放松肩膀，让紧张慢慢消散 ✨',
            '现在这一刻，你是安全的，你是被关爱的',
            '听听你的心跳，感受生命的节奏 💓'
        ],
        
        encouragement: [
            '你比你想象的更坚强！💪',
            '每个小进步都值得庆祝 🎉',
            '相信自己，你有无限的可能性',
            '困难只是暂时的，你的勇气是永恒的',
            '你的存在本身就很有意义 🌟',
            '今天的你已经很努力了！'
        ],
        
        mindfulness: [
            '此刻，专注于当下的感受 🧘‍♀️',
            '观察你的呼吸，不需要改变什么',
            '感受脚踏实地的稳定感',
            '注意周围的声音，让心灵安静下来',
            '你的思绪像云朵一样，让它们自然飘过'
        ],
        
        selfCare: [
            '记得照顾好自己，你值得被温柔对待 🌺',
            '今天有没有做一件让自己开心的事？',
            '喝杯温水，给身体一些关爱 💧',
            '适当的休息不是懒惰，是必需的',
            '对自己说句鼓励的话吧！',
            '你今天已经做得很好了 ✨'
        ],
        
        walking: [
            '我要去散个步，运动对心情很有帮助哦~',
            '走走走，一起来活动一下身体！',
            '让我到处看看，探索新的美好！',
            '散步能让心情变得更好呢！'
        ],
        
        jumping: [
            '跳一跳，心情也会跟着轻松起来！✨',
            '看我跳得多高！运动真开心！',
            '蹦蹦跳跳，烦恼都跳走了！',
            '耶！感受这份活力！'
        ],
        
        clicked: [
            '需要聊聊吗？我在这里倾听 👂',
            '想要一个温暖的拥抱吗？🤗',
            '告诉我你现在的感受吧',
            '有什么我可以帮助你的吗？',
            '要不要一起做个放松练习？'
        ],
        
        idle: [
            '记得关爱自己哦 💕',
            '深呼吸，感受当下的平静',
            '你今天做得很棒！',
            '要不要聊聊心情？',
            '我在这里陪伴你 🌙',
            '给自己一个微笑吧！😊'
        ],
        
        dragged: [
            '哇！带我去新地方！',
            '这样移动好有趣！',
            '我们要去哪里呢？',
            '谢谢你带我走走~'
        ]
    };
    
    // ==================== 样式定义 ====================
    function createStyles() {
        const style = document.createElement('style');
        style.textContent = `
            /* 心理健康助手容器 */
            .mental-health-bot {
                position: fixed;
                width: ${CONFIG.botSize}px;
                height: ${CONFIG.botSize}px;
                z-index: 999999;
                cursor: pointer;
                user-select: none;
                transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                filter: drop-shadow(0 8px 16px rgba(0,0,0,0.1));
            }
            
            .mental-health-bot:hover {
                transform: scale(1.05);
                filter: drop-shadow(0 12px 24px rgba(0,0,0,0.15));
            }
            
            .mental-health-bot.dragging {
                cursor: grabbing;
                transform: scale(1.1);
                filter: drop-shadow(0 16px 32px rgba(0,0,0,0.2));
            }
            
            .mental-health-bot.flipped {
                transform: scaleX(-1);
            }
            
            .mental-health-bot.flipped.dragging {
                transform: scaleX(-1) scale(1.1);
            }
            
            /* 机器人身体容器 */
            .bot-body {
                position: absolute;
                width: 100%;
                height: 100%;
                display: flex;
                flex-direction: column;
                align-items: center;
                animation: gentleFloat 4s ease-in-out infinite;
            }
            
            @keyframes gentleFloat {
                0%, 100% { transform: translateY(0px); }
                50% { transform: translateY(-6px); }
            }
            
            /* 天线 - 心理健康主题 */
            .bot-antenna {
                width: 3px;
                height: 20px;
                background: linear-gradient(to bottom, #FF69B4, #FFB6C1);
                position: relative;
                margin: 0 auto 5px;
                animation: antenna-sway 3s ease-in-out infinite;
            }
            
            .bot-antenna::before {
                content: '💙';
                position: absolute;
                top: -15px;
                left: 50%;
                transform: translateX(-50%);
                font-size: 12px;
                animation: heart-pulse 2s ease-in-out infinite;
            }
            
            @keyframes antenna-sway {
                0%, 100% { transform: rotate(0deg); }
                25% { transform: rotate(-5deg); }
                75% { transform: rotate(5deg); }
            }
            
            @keyframes heart-pulse {
                0%, 100% { transform: translateX(-50%) scale(1); }
                50% { transform: translateX(-50%) scale(1.2); }
            }
            
            /* 头部 - 温暖的粉色系 */
            .bot-head {
                width: 70px;
                height: 70px;
                background: linear-gradient(145deg, #FFE4E1 0%, #FFC0CB 50%, #FFB6C1 100%);
                border-radius: 50%;
                position: relative;
                animation: head-breathe 3s ease-in-out infinite;
                box-shadow: 
                    0 4px 12px rgba(255, 182, 193, 0.4),
                    inset -2px -2px 8px rgba(0, 0, 0, 0.1),
                    inset 2px 2px 8px rgba(255, 255, 255, 0.6);
            }
            
            @keyframes head-breathe {
                0%, 100% { transform: translateY(0) scale(1); }
                50% { transform: translateY(-2px) scale(1.02); }
            }
            
            /* 眼睛 - 温柔有神 */
            .bot-eyes {
                position: absolute;
                top: 22px;
                left: 50%;
                transform: translateX(-50%);
                display: flex;
                gap: 16px;
            }
            
            .bot-eye {
                width: 16px;
                height: 16px;
                background: white;
                border-radius: 50%;
                position: relative;
                box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            }
            
            .bot-pupil {
                position: absolute;
                width: 8px;
                height: 8px;
                background: #4A5568;
                border-radius: 50%;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                transition: all 0.2s ease;
            }
            
            .bot-pupil::after {
                content: '';
                position: absolute;
                top: 1px;
                left: 1px;
                width: 3px;
                height: 3px;
                background: white;
                border-radius: 50%;
            }
            
            /* 腮红 - 可爱的粉色 */
            .bot-cheek {
                position: absolute;
                top: 38px;
                width: 12px;
                height: 8px;
                background: rgba(255, 105, 180, 0.4);
                border-radius: 50%;
            }
            
            .bot-cheek.left { left: 8px; }
            .bot-cheek.right { right: 8px; }
            
            /* 嘴巴 - 温暖的微笑 */
            .bot-mouth {
                position: absolute;
                bottom: 18px;
                left: 50%;
                transform: translateX(-50%);
                width: 24px;
                height: 12px;
                border: 2px solid #FF69B4;
                border-top: none;
                border-radius: 0 0 12px 12px;
                transition: all 0.3s ease;
            }
            
            .bot-mouth.happy {
                border-color: #FF69B4;
                animation: smile-glow 2s ease-in-out infinite;
            }
            
            .bot-mouth.calm {
                border-color: #87CEEB;
                width: 20px;
                height: 10px;
            }
            
            .bot-mouth.concerned {
                border-color: #FFA07A;
                border-radius: 12px 12px 0 0;
                border-top: 2px solid #FFA07A;
                border-bottom: none;
            }
            
            @keyframes smile-glow {
                0%, 100% { box-shadow: 0 0 5px rgba(255, 105, 180, 0.3); }
                50% { box-shadow: 0 0 10px rgba(255, 105, 180, 0.6); }
            }
            
            /* 身体 - 圆润温暖 */
            .bot-torso {
                width: 60px;
                height: 50px;
                background: linear-gradient(145deg, #FFE4E1 0%, #FFC0CB 50%, #FFB6C1 100%);
                border-radius: 15px 15px 25px 25px;
                margin-top: 3px;
                position: relative;
                box-shadow: 
                    0 4px 12px rgba(255, 182, 193, 0.4),
                    inset -2px -2px 8px rgba(0, 0, 0, 0.1),
                    inset 2px 2px 8px rgba(255, 255, 255, 0.6);
            }
            
            /* 心形能量核心 */
            .bot-heart-core {
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                width: 20px;
                height: 20px;
                animation: heart-beat 1.5s ease-in-out infinite;
            }
            
            .bot-heart-core::before,
            .bot-heart-core::after {
                content: '';
                width: 10px;
                height: 16px;
                position: absolute;
                left: 10px;
                transform: rotate(-45deg);
                background: linear-gradient(45deg, #FF69B4, #FFB6C1);
                border-radius: 10px 10px 0 0;
                transform-origin: 0 100%;
                box-shadow: 0 0 10px rgba(255, 105, 180, 0.5);
            }
            
            .bot-heart-core::after {
                left: 0;
                transform: rotate(45deg);
                transform-origin: 100% 100%;
            }
            
            @keyframes heart-beat {
                0%, 100% { transform: translate(-50%, -50%) scale(1); }
                50% { transform: translate(-50%, -50%) scale(1.1); }
            }
            
            /* 手臂 - 可爱的小手 */
            .bot-arms {
                position: absolute;
                top: 8px;
                width: 100%;
                display: flex;
                justify-content: space-between;
                padding: 0 3px;
            }
            
            .bot-arm {
                width: 10px;
                height: 28px;
                background: linear-gradient(to bottom, #FFE4E1, #FFC0CB);
                border-radius: 5px;
                position: relative;
                animation: arm-gentle-swing 2s ease-in-out infinite;
                box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            }
            
            .bot-arm::after {
                content: '';
                position: absolute;
                bottom: -5px;
                left: 50%;
                transform: translateX(-50%);
                width: 12px;
                height: 12px;
                background: #FFC0CB;
                border-radius: 50%;
                box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            }
            
            .bot-arm.left {
                transform-origin: top center;
            }
            
            .bot-arm.right {
                transform-origin: top center;
                animation-delay: 1s;
            }
            
            .bot-arm.waving {
                animation: arm-wave 0.6s ease-in-out infinite;
            }
            
            @keyframes arm-gentle-swing {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-8deg); }
            }
            
            @keyframes arm-wave {
                0%, 100% { transform: rotate(-15deg); }
                50% { transform: rotate(15deg); }
            }
            
            /* 腿 - 稳定可爱 */
            .bot-legs {
                display: flex;
                gap: 12px;
                margin-top: 3px;
            }
            
            .bot-leg {
                width: 14px;
                height: 22px;
                background: linear-gradient(to bottom, #FFE4E1, #FFC0CB);
                border-radius: 7px;
                position: relative;
                box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            }
            
            .bot-foot {
                position: absolute;
                bottom: -6px;
                left: 50%;
                transform: translateX(-50%);
                width: 20px;
                height: 10px;
                background: #FFB6C1;
                border-radius: 5px;
                box-shadow: 0 2px 6px rgba(0, 0, 0, 0.2);
            }
            
            .bot-foot::before {
                content: '';
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                width: 14px;
                height: 4px;
                background: rgba(255, 255, 255, 0.3);
                border-radius: 2px;
            }
            
            /* 对话气泡 */
            .dialog-bubble {
                position: absolute;
                bottom: 110%;
                left: 50%;
                transform: translateX(-50%) translateY(10px);
                background: linear-gradient(135deg, #FFFFFF 0%, #FFF8F8 100%);
                color: #2D3748;
                padding: 12px 16px;
                border-radius: 20px;
                font-size: 14px;
                font-weight: 500;
                max-width: 200px;
                text-align: center;
                box-shadow: 
                    0 8px 24px rgba(0,0,0,0.1),
                    inset 0 1px 0 rgba(255,255,255,0.6);
                border: 2px solid rgba(255, 182, 193, 0.3);
                opacity: 0;
                transition: all 0.4s cubic-bezier(0.68, -0.55, 0.265, 1.55);
                pointer-events: none;
                z-index: 1000000;
                line-height: 1.4;
                margin-bottom: 8px;
            }
            
            .dialog-bubble::after {
                content: '';
                position: absolute;
                top: 100%;
                left: 50%;
                transform: translateX(-50%);
                width: 0;
                height: 0;
                border-left: 10px solid transparent;
                border-right: 10px solid transparent;
                border-top: 10px solid #FFFFFF;
                filter: drop-shadow(0 2px 4px rgba(0,0,0,0.1));
            }
            
            .dialog-bubble.visible {
                opacity: 1;
                transform: translateX(-50%) translateY(0);
            }
            
            /* 走路动画 */
            .mental-health-bot.walking .bot-leg:nth-child(1) {
                animation: leg-walk-left 0.8s ease-in-out infinite;
            }
            
            .mental-health-bot.walking .bot-leg:nth-child(2) {
                animation: leg-walk-right 0.8s ease-in-out infinite;
            }
            
            .mental-health-bot.walking .bot-arm.left {
                animation: arm-walk-left 0.8s ease-in-out infinite;
            }
            
            .mental-health-bot.walking .bot-arm.right {
                animation: arm-walk-right 0.8s ease-in-out infinite;
            }
            
            @keyframes leg-walk-left {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-12deg); }
            }
            
            @keyframes leg-walk-right {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(12deg); }
            }
            
            @keyframes arm-walk-left {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(10deg); }
            }
            
            @keyframes arm-walk-right {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-10deg); }
            }
            
            /* 跳跃动画 */
            .mental-health-bot.jumping {
                animation: bot-jump 1s ease-in-out;
            }
            
            @keyframes bot-jump {
                0%, 100% { transform: translateY(0); }
                50% { transform: translateY(-60px); }
            }
            
            /* 思考动画 */
            .mental-health-bot.thinking .bot-head {
                animation: head-think 1.2s ease-in-out infinite;
            }
            
            @keyframes head-think {
                0%, 100% { transform: translateY(0) rotate(0deg); }
                25% { transform: translateY(-2px) rotate(-2deg); }
                75% { transform: translateY(-2px) rotate(2deg); }
            }
            
            /* 情绪粒子效果 */
            .emotion-particle {
                position: absolute;
                font-size: 16px;
                pointer-events: none;
                animation: particle-float 2s ease-out forwards;
            }
            
            @keyframes particle-float {
                0% {
                    transform: translateY(0) scale(0);
                    opacity: 0;
                }
                20% {
                    opacity: 1;
                    transform: translateY(-10px) scale(1);
                }
                100% {
                    transform: translateY(-50px) scale(0.5);
                    opacity: 0;
                }
            }
            
            /* 响应式设计 */
            @media (max-width: 768px) {
                .mental-health-bot {
                    width: ${CONFIG.botSize * 0.8}px;
                    height: ${CONFIG.botSize * 0.8}px;
                }
                
                .dialog-bubble {
                    max-width: 160px;
                    font-size: 13px;
                    padding: 10px 14px;
                }
            }
            
            /* 无障碍支持 */
            .mental-health-bot:focus {
                outline: 3px solid #FF69B4;
                outline-offset: 4px;
            }
            
            /* 深色模式支持 */
            @media (prefers-color-scheme: dark) {
                .dialog-bubble {
                    background: linear-gradient(135deg, #2D3748 0%, #4A5568 100%);
                    color: #F7FAFC;
                    border-color: rgba(255, 182, 193, 0.3);
                }
                
                .dialog-bubble::after {
                    border-top-color: #2D3748;
                }
            }
        `;
        document.head.appendChild(style);
    }
    
    // ==================== 机器人创建 ====================
    function createBot() {
        const bot = document.createElement('div');
        bot.className = 'mental-health-bot';
        bot.setAttribute('role', 'button');
        bot.setAttribute('tabindex', '0');
        bot.setAttribute('aria-label', '心理健康助手小暖，点击互动');
        
        // 创建机器人结构 - 避免使用innerHTML
        const botBody = document.createElement('div');
        botBody.className = 'bot-body';
        
        // 对话气泡
        const dialogBubble = document.createElement('div');
        dialogBubble.className = 'dialog-bubble';
        botBody.appendChild(dialogBubble);
        
        // 天线
        const antenna = document.createElement('div');
        antenna.className = 'bot-antenna';
        botBody.appendChild(antenna);
        
        // 头部
        const head = document.createElement('div');
        head.className = 'bot-head';
        
        // 眼睛容器
        const eyes = document.createElement('div');
        eyes.className = 'bot-eyes';
        
        // 左眼
        const leftEye = document.createElement('div');
        leftEye.className = 'bot-eye';
        const leftPupil = document.createElement('div');
        leftPupil.className = 'bot-pupil';
        leftPupil.id = 'bot-pupil-left';
        leftEye.appendChild(leftPupil);
        eyes.appendChild(leftEye);
        
        // 右眼
        const rightEye = document.createElement('div');
        rightEye.className = 'bot-eye';
        const rightPupil = document.createElement('div');
        rightPupil.className = 'bot-pupil';
        rightPupil.id = 'bot-pupil-right';
        rightEye.appendChild(rightPupil);
        eyes.appendChild(rightEye);
        
        head.appendChild(eyes);
        
        // 腮红
        const leftCheek = document.createElement('div');
        leftCheek.className = 'bot-cheek left';
        head.appendChild(leftCheek);
        
        const rightCheek = document.createElement('div');
        rightCheek.className = 'bot-cheek right';
        head.appendChild(rightCheek);
        
        // 嘴巴
        const mouth = document.createElement('div');
        mouth.className = 'bot-mouth happy';
        head.appendChild(mouth);
        
        botBody.appendChild(head);
        
        // 身体
        const torso = document.createElement('div');
        torso.className = 'bot-torso';
        
        // 心形核心
        const heartCore = document.createElement('div');
        heartCore.className = 'bot-heart-core';
        torso.appendChild(heartCore);
        
        // 手臂容器
        const arms = document.createElement('div');
        arms.className = 'bot-arms';
        
        const leftArm = document.createElement('div');
        leftArm.className = 'bot-arm left';
        arms.appendChild(leftArm);
        
        const rightArm = document.createElement('div');
        rightArm.className = 'bot-arm right';
        arms.appendChild(rightArm);
        
        torso.appendChild(arms);
        botBody.appendChild(torso);
        
        // 腿部容器
        const legs = document.createElement('div');
        legs.className = 'bot-legs';
        
        // 左腿
        const leftLeg = document.createElement('div');
        leftLeg.className = 'bot-leg';
        const leftFoot = document.createElement('div');
        leftFoot.className = 'bot-foot';
        leftLeg.appendChild(leftFoot);
        legs.appendChild(leftLeg);
        
        // 右腿
        const rightLeg = document.createElement('div');
        rightLeg.className = 'bot-leg';
        const rightFoot = document.createElement('div');
        rightFoot.className = 'bot-foot';
        rightLeg.appendChild(rightFoot);
        legs.appendChild(rightLeg);
        
        botBody.appendChild(legs);
        bot.appendChild(botBody);
        
        // 设置初始位置
        updatePosition(bot);
        
        return bot;
    }
    
    // ==================== 位置管理 ====================
    function updatePosition(bot) {
        // 确保机器人在屏幕范围内
        state.x = Math.max(0, Math.min(window.innerWidth - CONFIG.botSize, state.x));
        state.y = Math.max(0, Math.min(window.innerHeight - CONFIG.botSize, state.y));
        
        bot.style.left = state.x + 'px';
        bot.style.top = state.y + 'px';
        
        // 保存位置到本地存储
        localStorage.setItem('mentalHealthBot_position', JSON.stringify({
            x: state.x,
            y: state.y
        }));
    }
    
    // ==================== 移动系统 ====================
    function startWalking(bot) {
        if (state.isWalking || state.isDragging) return;
        
        state.isWalking = true;
        bot.classList.add('walking');
        
        // 随机选择目标位置
        state.targetX = Math.random() * (window.innerWidth - CONFIG.botSize);
        state.targetY = Math.random() * (window.innerHeight - CONFIG.botSize);
        
        // 计算移动方向
        const deltaX = state.targetX - state.x;
        const deltaY = state.targetY - state.y;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        if (distance > 10) {
            state.velocityX = (deltaX / distance) * CONFIG.walkingSpeed;
            state.velocityY = (deltaY / distance) * CONFIG.walkingSpeed;
            
            // 设置朝向
            if (deltaX > 0) {
                bot.classList.remove('flipped');
                state.facingRight = true;
            } else {
                bot.classList.add('flipped');
                state.facingRight = false;
            }
            
            // 显示走路对话
            const walkMessage = getRandomDialog('walking');
            showDialog(bot, walkMessage);
            
            moveBot(bot);
        } else {
            stopWalking(bot);
        }
    }
    
    function moveBot(bot) {
        if (!state.isWalking) return;
        
        state.x += state.velocityX;
        state.y += state.velocityY;
        
        // 检查是否到达目标
        const deltaX = state.targetX - state.x;
        const deltaY = state.targetY - state.y;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        if (distance < 5) {
            stopWalking(bot);
            return;
        }
        
        updatePosition(bot);
        requestAnimationFrame(() => moveBot(bot));
    }
    
    function stopWalking(bot) {
        state.isWalking = false;
        state.velocityX = 0;
        state.velocityY = 0;
        bot.classList.remove('walking');
    }
    
    function jump(bot) {
        if (state.isWalking) return;
        
        bot.classList.add('jumping');
        const jumpMessage = getRandomDialog('jumping');
        showDialog(bot, jumpMessage);
        
        // 创建跳跃粒子效果
        createEmotionParticles(bot, ['✨', '💫', '⭐']);
        
        // 使用requestAnimationFrame替代setTimeout
        let jumpTimer = 0;
        function removeJumpClass() {
            jumpTimer++;
            if (jumpTimer >= 48) { // 约800ms (48 * 16.67ms)
                bot.classList.remove('jumping');
            } else {
                requestAnimationFrame(removeJumpClass);
            }
        }
        requestAnimationFrame(removeJumpClass);
    }
    
    // ==================== 对话系统 ====================
    function showDialog(bot, message, duration = CONFIG.dialogDuration) {
        const bubble = bot.querySelector('.dialog-bubble');
        if (!bubble) return;
        
        // 隐藏当前对话
        hideDialog(bot);
        
        // 使用requestAnimationFrame替代setTimeout
        let showTimer = 0;
        function showBubble() {
            showTimer++;
            if (showTimer >= 6) { // 约100ms (6 * 16.67ms)
                bubble.textContent = message;
                bubble.classList.add('visible');
                state.dialogVisible = true;
                state.currentDialog = message;
                
                // 自动隐藏
                let hideTimer = 0;
                const hideFrames = Math.floor(duration / 16.67); // 转换为帧数
                function hideBubble() {
                    hideTimer++;
                    if (hideTimer >= hideFrames) {
                        hideDialog(bot);
                    } else {
                        requestAnimationFrame(hideBubble);
                    }
                }
                requestAnimationFrame(hideBubble);
            } else {
                requestAnimationFrame(showBubble);
            }
        }
        requestAnimationFrame(showBubble);
    }
    
    function hideDialog(bot) {
        const bubble = bot.querySelector('.dialog-bubble');
        if (bubble) {
            bubble.classList.remove('visible');
            state.dialogVisible = false;
        }
    }
    
    function getRandomDialog(category) {
        const dialogs = DIALOGS[category] || DIALOGS.idle;
        return dialogs[Math.floor(Math.random() * dialogs.length)];
    }
    
    // ==================== 情绪表达 ====================
    function changeEmotion(bot, emotion) {
        const mouth = bot.querySelector('.bot-mouth');
        if (mouth) {
            mouth.className = `bot-mouth ${emotion}`;
        }
        state.currentEmotion = emotion;
    }
    
    function createEmotionParticles(bot, emojis) {
        const container = bot.querySelector('.bot-body');
        
        emojis.forEach((emoji, index) => {
            // 使用requestAnimationFrame替代setTimeout
            let delayTimer = 0;
            const delayFrames = Math.floor((index * 200) / 16.67); // 转换为帧数
            
            function createParticle() {
                delayTimer++;
                if (delayTimer >= delayFrames) {
                    const particle = document.createElement('div');
                    particle.className = 'emotion-particle';
                    particle.textContent = emoji;
                    particle.style.left = (Math.random() * 60 + 20) + 'px';
                    particle.style.top = '50px';
                    
                    container.appendChild(particle);
                    
                    // 2秒后移除粒子
                    let removeTimer = 0;
                    const removeFrames = Math.floor(2000 / 16.67); // 2秒转换为帧数
                    function removeParticle() {
                        removeTimer++;
                        if (removeTimer >= removeFrames) {
                            if (particle.parentNode) {
                                particle.parentNode.removeChild(particle);
                            }
                        } else {
                            requestAnimationFrame(removeParticle);
                        }
                    }
                    requestAnimationFrame(removeParticle);
                } else {
                    requestAnimationFrame(createParticle);
                }
            }
            requestAnimationFrame(createParticle);
        });
    }
    
    // ==================== 心理健康工具 ====================
    function startBreathingExercise(bot) {
        changeEmotion(bot, 'calm');
        bot.classList.add('thinking');
        
        const steps = [
            '让我们一起做深呼吸练习 🌸',
            '慢慢吸气... 1... 2... 3... 4...',
            '屏住呼吸... 1... 2... 3... 4...',
            '慢慢呼气... 1... 2... 3... 4... 5... 6...',
            '很好！再来一次...',
            '感受呼吸带来的平静 ✨',
            '你做得很棒！感觉好一些了吗？'
        ];
        
        let currentStep = 0;
        
        function nextStep() {
            if (currentStep < steps.length) {
                showDialog(bot, steps[currentStep], 4000);
                currentStep++;
                
                // 使用requestAnimationFrame替代setTimeout
                let stepTimer = 0;
                const stepFrames = Math.floor(4000 / 16.67); // 4秒转换为帧数
                function waitForNextStep() {
                    stepTimer++;
                    if (stepTimer >= stepFrames) {
                        nextStep();
                    } else {
                        requestAnimationFrame(waitForNextStep);
                    }
                }
                requestAnimationFrame(waitForNextStep);
            } else {
                bot.classList.remove('thinking');
                createEmotionParticles(bot, ['💙', '🌸', '✨']);
            }
        }
        
        nextStep();
    }
    
    function startGratitudePractice(bot) {
        changeEmotion(bot, 'happy');
        
        const prompts = [
            '让我们做个感恩练习！想想今天让你感激的事情 🙏',
            '回忆一个温暖的时刻，感受那份美好 ✨',
            '想想一个对你很重要的人，感谢他们的存在 💕',
            '感受身边的小确幸，它们都很珍贵 🌟'
        ];
        
        const randomPrompt = prompts[Math.floor(Math.random() * prompts.length)];
        showDialog(bot, randomPrompt, 6000);
        
        // 使用requestAnimationFrame替代setTimeout
        let gratitudeTimer = 0;
        const gratitudeFrames = Math.floor(6000 / 16.67); // 6秒转换为帧数
        function showGratitudeMessage() {
            gratitudeTimer++;
            if (gratitudeTimer >= gratitudeFrames) {
                showDialog(bot, '感恩的心情能带来内心的平静和喜悦 ✨', 4000);
                createEmotionParticles(bot, ['💕', '🌟', '🙏']);
            } else {
                requestAnimationFrame(showGratitudeMessage);
            }
        }
        requestAnimationFrame(showGratitudeMessage);
    }
    
    // ==================== 交互处理 ====================
    function handleClick(bot, event) {
        event.preventDefault();
        
        state.interactionCount++;
        state.lastInteractionTime = Date.now();
        
        // 停止当前动作
        stopWalking(bot);
        
        // 根据交互次数选择不同的行为
        if (state.interactionCount === 1) {
            const message = getRandomDialog('greeting');
            showDialog(bot, message);
            changeEmotion(bot, 'happy');
            
            // 挥手欢迎
            const arms = bot.querySelectorAll('.bot-arm');
            arms.forEach(arm => arm.classList.add('waving'));
            
            // 使用requestAnimationFrame替代setTimeout
            let waveTimer = 0;
            const waveFrames = Math.floor(2000 / 16.67); // 2秒转换为帧数
            function stopWaving() {
                waveTimer++;
                if (waveTimer >= waveFrames) {
                    arms.forEach(arm => arm.classList.remove('waving'));
                } else {
                    requestAnimationFrame(stopWaving);
                }
            }
            requestAnimationFrame(stopWaving);
            
        } else if (state.interactionCount % 6 === 0) {
            // 每6次交互跳跃一次
            jump(bot);
            
        } else if (state.interactionCount % 4 === 0) {
            // 每4次交互提供心理健康工具
            const tools = ['breathing', 'gratitude'];
            const randomTool = tools[Math.floor(Math.random() * tools.length)];
            
            if (randomTool === 'breathing') {
                startBreathingExercise(bot);
            } else {
                startGratitudePractice(bot);
            }
            
        } else if (state.interactionCount % 3 === 0) {
            // 每3次交互开始走路
            startWalking(bot);
            
        } else {
            // 普通交互
            const categories = ['supportive', 'encouragement', 'mindfulness', 'selfCare'];
            const category = categories[Math.floor(Math.random() * categories.length)];
            const message = getRandomDialog(category);
            showDialog(bot, message);
            
            // 根据对话类型改变情绪
            if (category === 'supportive' || category === 'encouragement') {
                changeEmotion(bot, 'happy');
                createEmotionParticles(bot, ['💙', '✨', '🌟']);
            } else {
                changeEmotion(bot, 'calm');
            }
        }
        
        // 点击动画效果
        bot.style.transform = 'scale(0.95)';
        
        // 使用requestAnimationFrame替代setTimeout
        let clickTimer = 0;
        const clickFrames = Math.floor(150 / 16.67); // 150ms转换为帧数
        function resetTransform() {
            clickTimer++;
            if (clickTimer >= clickFrames) {
                bot.style.transform = '';
            } else {
                requestAnimationFrame(resetTransform);
            }
        }
        requestAnimationFrame(resetTransform);
    }
    
    // ==================== 拖拽功能 ====================
    function setupDragging(bot) {
        let isDragging = false;
        let dragOffset = { x: 0, y: 0 };
        
        function startDrag(e) {
            isDragging = true;
            state.isDragging = true;
            bot.classList.add('dragging');
            stopWalking(bot);
            
            const clientX = e.touches ? e.touches[0].clientX : e.clientX;
            const clientY = e.touches ? e.touches[0].clientY : e.clientY;
            
            dragOffset.x = clientX - state.x;
            dragOffset.y = clientY - state.y;
            
            document.body.style.userSelect = 'none';
            e.preventDefault();
        }
        
        function drag(e) {
            if (!isDragging) return;
            
            const clientX = e.touches ? e.touches[0].clientX : e.clientX;
            const clientY = e.touches ? e.touches[0].clientY : e.clientY;
            
            state.x = clientX - dragOffset.x;
            state.y = clientY - dragOffset.y;
            
            updatePosition(bot);
            e.preventDefault();
        }
        
        function endDrag() {
            if (isDragging) {
                isDragging = false;
                state.isDragging = false;
                bot.classList.remove('dragging');
                document.body.style.userSelect = '';
                
                // 拖拽结束后的温暖回应
                let dragTimer = 0;
                const dragFrames = Math.floor(500 / 16.67); // 500ms转换为帧数
                function showDragResponse() {
                    dragTimer++;
                    if (dragTimer >= dragFrames) {
                        if (!state.dialogVisible) {
                            const message = getRandomDialog('dragged');
                            showDialog(bot, message);
                            changeEmotion(bot, 'happy');
                        }
                    } else {
                        requestAnimationFrame(showDragResponse);
                    }
                }
                requestAnimationFrame(showDragResponse);
            }
        }
        
        // 鼠标事件
        bot.addEventListener('mousedown', startDrag);
        document.addEventListener('mousemove', drag);
        document.addEventListener('mouseup', endDrag);
        
        // 触摸事件
        bot.addEventListener('touchstart', startDrag, { passive: false });
        document.addEventListener('touchmove', drag, { passive: false });
        document.addEventListener('touchend', endDrag);
        
        // 点击事件（区分拖拽和点击）
        bot.addEventListener('click', (e) => {
            if (!state.isDragging) {
                handleClick(bot, e);
            }
        });
        
        // 键盘支持
        bot.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                handleClick(bot, e);
            }
        });
    }
    
    // ==================== 自动行为 ====================
    function startIdleBehavior(bot) {
        // 自动对话 - 使用requestAnimationFrame替代setInterval
        let idleTimer = 0;
        const idleFrames = Math.floor(CONFIG.idleDialogInterval / 16.67);
        
        function checkIdleDialog() {
            idleTimer++;
            if (idleTimer >= idleFrames) {
                if (!state.dialogVisible && !state.isDragging && !state.isWalking) {
                    const now = Date.now();
                    const timeSinceLastInteraction = now - state.lastInteractionTime;
                    
                    let category = 'idle';
                    if (timeSinceLastInteraction > 600000) { // 10分钟无交互
                        category = 'supportive';
                    } else if (timeSinceLastInteraction > 300000) { // 5分钟无交互
                        category = 'selfCare';
                    }
                    
                    const message = getRandomDialog(category);
                    showDialog(bot, message);
                    changeEmotion(bot, 'calm');
                }
                idleTimer = 0; // 重置计时器
            }
            requestAnimationFrame(checkIdleDialog);
        }
        requestAnimationFrame(checkIdleDialog);
        
        // 自动移动 - 使用requestAnimationFrame替代setInterval
        let moveTimer = 0;
        const moveFrames = Math.floor(CONFIG.autoMoveInterval / 16.67);
        
        function checkAutoMove() {
            moveTimer++;
            if (moveTimer >= moveFrames) {
                if (!state.isDragging && !state.dialogVisible && Math.random() < 0.3) {
                    if (Math.random() < 0.7) {
                        startWalking(bot);
                    } else {
                        jump(bot);
                    }
                }
                moveTimer = 0; // 重置计时器
            }
            requestAnimationFrame(checkAutoMove);
        }
        requestAnimationFrame(checkAutoMove);
    }
    
    // ==================== 窗口事件处理 ====================
    function handleWindowResize() {
        const bot = document.querySelector('.mental-health-bot');
        if (bot) {
            updatePosition(bot);
        }
    }
    
    // ==================== 初始化 ====================
    function init() {
        // 检查是否已经存在机器人
        if (document.querySelector('.mental-health-bot')) {
            console.log('心理健康助手已经存在');
            return;
        }
        
        // 创建样式
        createStyles();
        
        // 恢复保存的位置
        const savedPosition = localStorage.getItem('mentalHealthBot_position');
        if (savedPosition) {
            try {
                const pos = JSON.parse(savedPosition);
                state.x = pos.x;
                state.y = pos.y;
            } catch (e) {
                console.log('无法恢复保存的位置');
            }
        }
        
        // 创建机器人
        const bot = createBot();
        document.body.appendChild(bot);
        
        // 设置交互
        setupDragging(bot);
        
        // 启动自动行为
        startIdleBehavior(bot);
        
        // 窗口大小变化处理
        window.addEventListener('resize', handleWindowResize);
        
        // 显示欢迎消息
        let welcomeTimer = 0;
        const welcomeFrames = Math.floor(1000 / 16.67); // 1秒转换为帧数
        function showWelcome() {
            welcomeTimer++;
            if (welcomeTimer >= welcomeFrames) {
                const welcomeMessage = getRandomDialog('greeting');
                showDialog(bot, welcomeMessage, 6000);
                changeEmotion(bot, 'happy');
                
                // 欢迎挥手
                const arms = bot.querySelectorAll('.bot-arm');
                arms.forEach(arm => arm.classList.add('waving'));
                
                let welcomeWaveTimer = 0;
                const welcomeWaveFrames = Math.floor(3000 / 16.67); // 3秒转换为帧数
                function stopWelcomeWaving() {
                    welcomeWaveTimer++;
                    if (welcomeWaveTimer >= welcomeWaveFrames) {
                        arms.forEach(arm => arm.classList.remove('waving'));
                    } else {
                        requestAnimationFrame(stopWelcomeWaving);
                    }
                }
                requestAnimationFrame(stopWelcomeWaving);
                
                createEmotionParticles(bot, ['💙', '🌟', '✨']);
            } else {
                requestAnimationFrame(showWelcome);
            }
        }
        requestAnimationFrame(showWelcome);
        
        console.log('心理健康助手小暖已启动 💙');
    }
    
    // ==================== 清理函数 ====================
    function cleanup() {
        const bot = document.querySelector('.mental-health-bot');
        if (bot) {
            bot.remove();
        }
        
        // 移除事件监听器
        window.removeEventListener('resize', handleWindowResize);
        
        console.log('心理健康助手已清理');
    }
    
    // ==================== 公共API ====================
    window.MentalHealthBot = {
        init: init,
        cleanup: cleanup,
        showMessage: function(message) {
            const bot = document.querySelector('.mental-health-bot');
            if (bot) {
                showDialog(bot, message);
            }
        },
        startWalking: function() {
            const bot = document.querySelector('.mental-health-bot');
            if (bot) {
                startWalking(bot);
            }
        },
        jump: function() {
            const bot = document.querySelector('.mental-health-bot');
            if (bot) {
                jump(bot);
            }
        },
        startBreathingExercise: function() {
            const bot = document.querySelector('.mental-health-bot');
            if (bot) {
                startBreathingExercise(bot);
            }
        },
        startGratitudePractice: function() {
            const bot = document.querySelector('.mental-health-bot');
            if (bot) {
                startGratitudePractice(bot);
            }
        }
    };
    
    // ==================== 自动启动 ====================
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    
})();