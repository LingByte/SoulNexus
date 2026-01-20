// 灵动机器人桌宠 V2.0
// 全新设计，更可爱更协调
// 作者: Manus AI

(function() {
    'use strict';
    
    // ==================== 配置 ====================
    const CONFIG = {
        robotName: 'Robo',
        robotSize: 140,
        moveSpeed: 2,
        idleDialogInterval: 15000,
        autoMoveInterval: 8000,
        dialogDuration: 4000
    };
    
    // ==================== 状态 ====================
    let state = {
        x: window.innerWidth - 200,
        y: window.innerHeight - 200,
        velocityX: 0,
        velocityY: 0,
        currentAction: 'idle',
        currentEmotion: 'happy',
        isMoving: false,
        isDragging: false,
        facingRight: true,
        eyeX: 0,
        eyeY: 0,
        dialogVisible: false,
        currentDialog: ''
    };
    
    // 对话库
    const DIALOGS = {
        greeting: [
            '嗨！我是Robo！👋',
            '你好呀！很高兴见到你！😊',
            '哇！有人来了！',
            '嘿嘿，我在这里哦！'
        ],
        idle: [
            '无聊...要不要一起玩？',
            '我在想什么好玩的事情...',
            '嗯...今天天气不错！',
            '你在干什么呢？',
            '我可以帮你做点什么吗？',
            '要不要聊聊天？',
            '我会很多技能哦！',
            '点击我试试看！'
        ],
        walking: [
            '我要去散个步~',
            '走走走，运动一下！',
            '让我到处看看！',
            '探索新地方！'
        ],
        jumping: [
            '跳！✨',
            '我会跳得很高！',
            '看我的跳跃！',
            '耶！'
        ],
        clicked: [
            '哎呀！你点我了！',
            '嘿嘿，找我有事吗？',
            '怎么啦？',
            '我在这里！',
            '需要帮忙吗？',
            '点我干嘛~'
        ],
        dragged: [
            '哇！带我飞！',
            '好快！',
            '我要晕了...',
            '慢一点啦！'
        ]
    };
    
    // ==================== 样式 ====================
    function createStyles() {
        const style = document.createElement('style');
        style.textContent = `
            /* 机器人容器 */
            .robot-pet-v2 {
                position: fixed;
                width: ${CONFIG.robotSize}px;
                height: ${CONFIG.robotSize}px;
                z-index: 999999;
                cursor: pointer;
                user-select: none;
                transition: transform 0.3s ease;
            }
            
            .robot-pet-v2.dragging {
                cursor: grabbing;
                transform: scale(1.1);
            }
            
            .robot-pet-v2.flipped {
                transform: scaleX(-1);
            }
            
            .robot-pet-v2.flipped.dragging {
                transform: scaleX(-1) scale(1.1);
            }
            
            /* 机器人身体 */
            .robot-body-v2 {
                position: absolute;
                width: 100%;
                height: 100%;
                display: flex;
                flex-direction: column;
                align-items: center;
                filter: drop-shadow(0 10px 25px rgba(0, 0, 0, 0.3));
            }
            
            /* 天线 */
            .robot-antenna-v2 {
                width: 2px;
                height: 25px;
                background: linear-gradient(to bottom, #ff6b6b, #ff8787);
                position: relative;
                margin: 0 auto 5px;
                animation: antenna-sway 3s ease-in-out infinite;
            }
            
            .robot-antenna-v2::before {
                content: '';
                position: absolute;
                top: -8px;
                left: 50%;
                transform: translateX(-50%);
                width: 10px;
                height: 10px;
                background: radial-gradient(circle, #ffd93d 0%, #ff6b6b 100%);
                border-radius: 50%;
                box-shadow: 0 0 20px rgba(255, 107, 107, 0.8);
                animation: antenna-blink 2s ease-in-out infinite;
            }
            
            @keyframes antenna-sway {
                0%, 100% { transform: rotate(0deg); }
                25% { transform: rotate(-8deg); }
                75% { transform: rotate(8deg); }
            }
            
            @keyframes antenna-blink {
                0%, 100% { opacity: 1; transform: translateX(-50%) scale(1); }
                50% { opacity: 0.6; transform: translateX(-50%) scale(0.9); }
            }
            
            /* 头部 - 圆润可爱 */
            .robot-head-v2 {
                width: 80px;
                height: 80px;
                background: linear-gradient(145deg, #4ecdc4 0%, #44a08d 100%);
                border-radius: 50%;
                position: relative;
                animation: head-float 2.5s ease-in-out infinite;
                box-shadow: 
                    0 5px 15px rgba(68, 160, 141, 0.4),
                    inset -3px -3px 10px rgba(0, 0, 0, 0.2),
                    inset 3px 3px 10px rgba(255, 255, 255, 0.3);
            }
            
            @keyframes head-float {
                0%, 100% { transform: translateY(0); }
                50% { transform: translateY(-5px); }
            }
            
            /* 眼睛 - 大而有神 */
            .robot-eyes-v2 {
                position: absolute;
                top: 25px;
                left: 50%;
                transform: translateX(-50%);
                display: flex;
                gap: 20px;
            }
            
            .robot-eye-v2 {
                width: 20px;
                height: 20px;
                background: white;
                border-radius: 50%;
                position: relative;
                box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2);
            }
            
            .robot-pupil-v2 {
                position: absolute;
                width: 10px;
                height: 10px;
                background: #2c3e50;
                border-radius: 50%;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                transition: all 0.2s ease;
            }
            
            .robot-pupil-v2::after {
                content: '';
                position: absolute;
                top: 2px;
                left: 2px;
                width: 4px;
                height: 4px;
                background: white;
                border-radius: 50%;
            }
            
            /* 腮红 */
            .robot-cheek-v2 {
                position: absolute;
                top: 45px;
                width: 15px;
                height: 10px;
                background: rgba(255, 107, 107, 0.4);
                border-radius: 50%;
            }
            
            .robot-cheek-v2.left {
                left: 8px;
            }
            
            .robot-cheek-v2.right {
                right: 8px;
            }
            
            /* 嘴巴 - 可爱的微笑 */
            .robot-mouth-v2 {
                position: absolute;
                bottom: 20px;
                left: 50%;
                transform: translateX(-50%);
                width: 30px;
                height: 15px;
                border: 3px solid #2c3e50;
                border-top: none;
                border-radius: 0 0 15px 15px;
                transition: all 0.3s ease;
            }
            
            .robot-mouth-v2.happy {
                border-radius: 0 0 15px 15px;
            }
            
            .robot-mouth-v2.sad {
                border-radius: 15px 15px 0 0;
                border-top: 3px solid #2c3e50;
                border-bottom: none;
            }
            
            .robot-mouth-v2.surprised {
                border-radius: 50%;
                border: 3px solid #2c3e50;
                width: 20px;
                height: 20px;
            }
            
            /* 身体 - 圆润的躯干 */
            .robot-torso-v2 {
                width: 70px;
                height: 60px;
                background: linear-gradient(145deg, #4ecdc4 0%, #44a08d 100%);
                border-radius: 20px 20px 30px 30px;
                margin-top: 5px;
                position: relative;
                box-shadow: 
                    0 5px 15px rgba(68, 160, 141, 0.4),
                    inset -3px -3px 10px rgba(0, 0, 0, 0.2),
                    inset 3px 3px 10px rgba(255, 255, 255, 0.3);
            }
            
            /* 能量核心 */
            .robot-core-v2 {
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                width: 25px;
                height: 25px;
                background: radial-gradient(circle, rgba(255, 255, 255, 0.6), rgba(78, 205, 196, 0.3));
                border-radius: 50%;
                border: 2px solid rgba(255, 255, 255, 0.5);
                animation: core-pulse 2s ease-in-out infinite;
            }
            
            @keyframes core-pulse {
                0%, 100% { 
                    box-shadow: 0 0 10px rgba(78, 205, 196, 0.5);
                    transform: translate(-50%, -50%) scale(1);
                }
                50% { 
                    box-shadow: 0 0 20px rgba(78, 205, 196, 0.8);
                    transform: translate(-50%, -50%) scale(1.1);
                }
            }
            
            /* 手臂 - 圆润连贯 */
            .robot-arms-v2 {
                position: absolute;
                top: 10px;
                width: 100%;
                display: flex;
                justify-content: space-between;
                padding: 0 5px;
            }
            
            .robot-arm-v2 {
                width: 12px;
                height: 35px;
                background: linear-gradient(to bottom, #4ecdc4, #44a08d);
                border-radius: 6px;
                position: relative;
                animation: arm-swing 1.5s ease-in-out infinite;
                box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2);
            }
            
            .robot-arm-v2::after {
                content: '';
                position: absolute;
                bottom: -6px;
                left: 50%;
                transform: translateX(-50%);
                width: 14px;
                height: 14px;
                background: #44a08d;
                border-radius: 50%;
                box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2);
            }
            
            .robot-arm-v2.left {
                transform-origin: top center;
            }
            
            .robot-arm-v2.right {
                transform-origin: top center;
                animation-delay: 0.75s;
            }
            
            .robot-arm-v2.waving {
                animation: arm-wave 0.5s ease-in-out infinite;
            }
            
            @keyframes arm-swing {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-10deg); }
            }
            
            @keyframes arm-wave {
                0%, 100% { transform: rotate(-20deg); }
                50% { transform: rotate(20deg); }
            }
            
            /* 腿 - 圆润可爱 */
            .robot-legs-v2 {
                display: flex;
                gap: 15px;
                margin-top: 5px;
            }
            
            .robot-leg-v2 {
                width: 16px;
                height: 25px;
                background: linear-gradient(to bottom, #4ecdc4, #44a08d);
                border-radius: 8px;
                position: relative;
                box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2);
            }
            
            .robot-foot-v2 {
                position: absolute;
                bottom: -8px;
                left: 50%;
                transform: translateX(-50%);
                width: 24px;
                height: 12px;
                background: #44a08d;
                border-radius: 6px;
                box-shadow: 0 3px 8px rgba(0, 0, 0, 0.3);
            }
            
            .robot-foot-v2::before {
                content: '';
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                width: 18px;
                height: 6px;
                background: rgba(255, 255, 255, 0.2);
                border-radius: 3px;
            }
            
            /* 对话气泡 */
            .robot-dialog-v2 {
                position: absolute;
                bottom: 100%;
                left: 50%;
                transform: translateX(-50%) translateY(-15px);
                background: white;
                color: #2c3e50;
                padding: 12px 18px;
                border-radius: 20px;
                box-shadow: 0 5px 20px rgba(0, 0, 0, 0.2);
                white-space: nowrap;
                font-size: 15px;
                font-weight: 600;
                opacity: 0;
                pointer-events: none;
                transition: all 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55);
                margin-bottom: 10px;
                max-width: 220px;
                white-space: normal;
                text-align: center;
            }
            
            .robot-dialog-v2.show {
                opacity: 1;
                transform: translateX(-50%) translateY(0);
            }
            
            .robot-dialog-v2::after {
                content: '';
                position: absolute;
                top: 100%;
                left: 50%;
                transform: translateX(-50%);
                width: 0;
                height: 0;
                border-left: 10px solid transparent;
                border-right: 10px solid transparent;
                border-top: 10px solid white;
            }
            
            /* 粒子效果 */
            .robot-particle-v2 {
                position: absolute;
                width: 6px;
                height: 6px;
                background: #4ecdc4;
                border-radius: 50%;
                pointer-events: none;
                animation: particle-float 1s ease-out forwards;
            }
            
            @keyframes particle-float {
                0% {
                    transform: translateY(0) scale(1);
                    opacity: 1;
                }
                100% {
                    transform: translateY(-30px) scale(0);
                    opacity: 0;
                }
            }
            
            /* 表情特效 */
            .robot-emoji-v2 {
                position: absolute;
                font-size: 28px;
                pointer-events: none;
                animation: emoji-float 2s ease-out forwards;
            }
            
            @keyframes emoji-float {
                0% {
                    transform: translateY(0) scale(0);
                    opacity: 0;
                }
                20% {
                    opacity: 1;
                    transform: translateY(-10px) scale(1);
                }
                100% {
                    transform: translateY(-60px) scale(0.5);
                    opacity: 0;
                }
            }
            
            /* 走路动画 */
            .robot-pet-v2.walking .robot-leg-v2:nth-child(1) {
                animation: leg-walk-left 0.6s ease-in-out infinite;
            }
            
            .robot-pet-v2.walking .robot-leg-v2:nth-child(2) {
                animation: leg-walk-right 0.6s ease-in-out infinite;
            }
            
            @keyframes leg-walk-left {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-15deg); }
            }
            
            @keyframes leg-walk-right {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(15deg); }
            }
            
            /* 跳跃动画 */
            .robot-pet-v2.jumping {
                animation: robot-jump 0.8s ease-in-out;
            }
            
            @keyframes robot-jump {
                0%, 100% { transform: translateY(0); }
                50% { transform: translateY(-70px); }
            }
            
            /* 思考动画 */
            .robot-pet-v2.thinking .robot-head-v2 {
                animation: head-think 1s ease-in-out infinite;
            }
            
            @keyframes head-think {
                0%, 100% { transform: translateY(0) rotate(0deg); }
                25% { transform: translateY(-3px) rotate(-3deg); }
                75% { transform: translateY(-3px) rotate(3deg); }
            }
        `;
        document.head.appendChild(style);
    }
    
    // ==================== HTML结构 ====================
    function createHTML() {
        const robot = document.createElement('div');
        robot.className = 'robot-pet-v2';
        robot.id = 'robot-pet-v2';
        robot.style.left = state.x + 'px';
        robot.style.top = state.y + 'px';
        
        robot.innerHTML = `
            <div class="robot-body-v2">
                <!-- 对话气泡 -->
                <div class="robot-dialog-v2" id="robot-dialog-v2"></div>
                
                <!-- 天线 -->
                <div class="robot-antenna-v2"></div>
                
                <!-- 头部 -->
                <div class="robot-head-v2">
                    <!-- 眼睛 -->
                    <div class="robot-eyes-v2">
                        <div class="robot-eye-v2">
                            <div class="robot-pupil-v2" id="robot-pupil-left-v2"></div>
                        </div>
                        <div class="robot-eye-v2">
                            <div class="robot-pupil-v2" id="robot-pupil-right-v2"></div>
                        </div>
                    </div>
                    
                    <!-- 腮红 -->
                    <div class="robot-cheek-v2 left"></div>
                    <div class="robot-cheek-v2 right"></div>
                    
                    <!-- 嘴巴 -->
                    <div class="robot-mouth-v2 happy" id="robot-mouth-v2"></div>
                </div>
                
                <!-- 躯干 -->
                <div class="robot-torso-v2">
                    <div class="robot-core-v2"></div>
                    <div class="robot-arms-v2">
                        <div class="robot-arm-v2 left"></div>
                        <div class="robot-arm-v2 right"></div>
                    </div>
                </div>
                
                <!-- 腿 -->
                <div class="robot-legs-v2">
                    <div class="robot-leg-v2">
                        <div class="robot-foot-v2"></div>
                    </div>
                    <div class="robot-leg-v2">
                        <div class="robot-foot-v2"></div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(robot);
    }
    
    // ==================== 功能函数 ====================
    
    // 显示对话
    function showDialog(text) {
        const dialog = document.getElementById('robot-dialog-v2');
        if (!dialog) return;
        
        dialog.textContent = text;
        dialog.classList.add('show');
        state.dialogVisible = true;
        state.currentDialog = text;
        
        setTimeout(() => {
            dialog.classList.remove('show');
            state.dialogVisible = false;
        }, CONFIG.dialogDuration);
    }
    
    // 随机对话
    function randomDialog(category = 'idle') {
        const dialogs = DIALOGS[category] || DIALOGS.idle;
        const text = dialogs[Math.floor(Math.random() * dialogs.length)];
        showDialog(text);
    }
    
    // 设置表情
    function setEmotion(emotion) {
        state.currentEmotion = emotion;
        const mouth = document.getElementById('robot-mouth-v2');
        if (!mouth) return;
        
        mouth.className = `robot-mouth-v2 ${emotion}`;
    }
    
    // 创建粒子
    function createParticle(x, y) {
        const robot = document.getElementById('robot-pet-v2');
        if (!robot) return;
        
        const particle = document.createElement('div');
        particle.className = 'robot-particle-v2';
        particle.style.left = x + 'px';
        particle.style.bottom = y + 'px';
        robot.appendChild(particle);
        
        setTimeout(() => particle.remove(), 1000);
    }
    
    // 创建表情符号
    function createEmoji(emoji) {
        const robot = document.getElementById('robot-pet-v2');
        if (!robot) return;
        
        const emojiEl = document.createElement('div');
        emojiEl.className = 'robot-emoji-v2';
        emojiEl.textContent = emoji;
        emojiEl.style.left = '50%';
        emojiEl.style.top = '-30px';
        emojiEl.style.transform = 'translateX(-50%)';
        robot.appendChild(emojiEl);
        
        setTimeout(() => emojiEl.remove(), 2000);
    }
    
    // 执行动作
    function performAction(action) {
        const robot = document.getElementById('robot-pet-v2');
        if (!robot) return;
        
        robot.classList.remove('walking', 'jumping', 'thinking');
        
        state.currentAction = action;
        
        switch(action) {
            case 'walking':
                robot.classList.add('walking');
                randomDialog('walking');
                break;
                
            case 'jumping':
                robot.classList.add('jumping');
                randomDialog('jumping');
                createEmoji('✨');
                setEmotion('happy');
                setTimeout(() => {
                    robot.classList.remove('jumping');
                    setEmotion('happy');
                }, 800);
                break;
                
            case 'waving':
                const arms = robot.querySelectorAll('.robot-arm-v2');
                arms.forEach(arm => arm.classList.add('waving'));
                randomDialog('greeting');
                createEmoji('👋');
                setTimeout(() => {
                    arms.forEach(arm => arm.classList.remove('waving'));
                }, 2000);
                break;
                
            case 'thinking':
                robot.classList.add('thinking');
                setEmotion('thinking');
                createEmoji('💭');
                setTimeout(() => {
                    robot.classList.remove('thinking');
                    setEmotion('happy');
                }, 3000);
                break;
        }
    }
    
    // 移动机器人
    function moveRobot(targetX, targetY) {
        const robot = document.getElementById('robot-pet-v2');
        if (!robot || state.isDragging) return;
        
        const deltaX = targetX - state.x;
        const deltaY = targetY - state.y;
        const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
        
        if (distance < 10) {
            state.isMoving = false;
            robot.classList.remove('walking');
            return;
        }
        
        state.isMoving = true;
        robot.classList.add('walking');
        
        if (deltaX > 0) {
            state.facingRight = true;
            robot.classList.remove('flipped');
        } else {
            state.facingRight = false;
            robot.classList.add('flipped');
        }
        
        state.velocityX = (deltaX / distance) * CONFIG.moveSpeed;
        state.velocityY = (deltaY / distance) * CONFIG.moveSpeed;
        
        state.x += state.velocityX;
        state.y += state.velocityY;
        
        robot.style.left = state.x + 'px';
        robot.style.top = state.y + 'px';
        
        if (Math.random() < 0.3) {
            createParticle(CONFIG.robotSize / 2, 0);
        }
    }
    
    // 随机移动
    function randomMove() {
        if (state.isDragging || state.isMoving) return;
        
        const targetX = Math.random() * (window.innerWidth - CONFIG.robotSize);
        const targetY = Math.random() * (window.innerHeight - CONFIG.robotSize);
        
        performAction('walking');
        
        const moveInterval = setInterval(() => {
            moveRobot(targetX, targetY);
            
            if (!state.isMoving) {
                clearInterval(moveInterval);
                performAction('idle');
            }
        }, 50);
    }
    
    // 眼睛跟随鼠标
    function updateEyes(mouseX, mouseY) {
        const robot = document.getElementById('robot-pet-v2');
        if (!robot) return;
        
        const robotRect = robot.getBoundingClientRect();
        const robotCenterX = robotRect.left + robotRect.width / 2;
        const robotCenterY = robotRect.top + 40;
        
        const angle = Math.atan2(mouseY - robotCenterY, mouseX - robotCenterX);
        const distance = Math.min(3, Math.sqrt(
            Math.pow(mouseX - robotCenterX, 2) + 
            Math.pow(mouseY - robotCenterY, 2)
        ) / 50);
        
        const eyeX = Math.cos(angle) * distance;
        const eyeY = Math.sin(angle) * distance;
        
        const pupils = robot.querySelectorAll('.robot-pupil-v2');
        pupils.forEach(pupil => {
            pupil.style.transform = `translate(calc(-50% + ${eyeX}px), calc(-50% + ${eyeY}px))`;
        });
    }
    
    // ==================== 事件处理 ====================
    function bindEvents() {
        const robot = document.getElementById('robot-pet-v2');
        if (!robot) return;
        
        let dragStartX, dragStartY;
        
        robot.addEventListener('mousedown', (e) => {
            state.isDragging = true;
            robot.classList.add('dragging');
            dragStartX = e.clientX - state.x;
            dragStartY = e.clientY - state.y;
            e.preventDefault();
        });
        
        document.addEventListener('mousemove', (e) => {
            updateEyes(e.clientX, e.clientY);
            
            if (state.isDragging) {
                state.x = e.clientX - dragStartX;
                state.y = e.clientY - dragStartY;
                
                state.x = Math.max(0, Math.min(window.innerWidth - CONFIG.robotSize, state.x));
                state.y = Math.max(0, Math.min(window.innerHeight - CONFIG.robotSize, state.y));
                
                robot.style.left = state.x + 'px';
                robot.style.top = state.y + 'px';
            }
        });
        
        document.addEventListener('mouseup', () => {
            if (state.isDragging) {
                state.isDragging = false;
                robot.classList.remove('dragging');
                randomDialog('dragged');
            }
        });
        
        robot.addEventListener('click', (e) => {
            if (!state.isDragging) {
                const actions = ['jumping', 'waving', 'thinking'];
                const randomAction = actions[Math.floor(Math.random() * actions.length)];
                performAction(randomAction);
                randomDialog('clicked');
            }
        });
        
        robot.addEventListener('dblclick', () => {
            randomMove();
        });
    }
    
    // ==================== 自动行为 ====================
    function startAutoBehavior() {
        setInterval(() => {
            if (!state.dialogVisible && Math.random() < 0.5) {
                randomDialog('idle');
            }
        }, CONFIG.idleDialogInterval);
        
        setInterval(() => {
            if (Math.random() < 0.3) {
                randomMove();
            }
        }, CONFIG.autoMoveInterval);
        
        setInterval(() => {
            if (!state.isMoving && Math.random() < 0.2) {
                const actions = ['waving', 'thinking', 'jumping'];
                const randomAction = actions[Math.floor(Math.random() * actions.length)];
                performAction(randomAction);
            }
        }, 10000);
    }
    
    // ==================== 初始化 ====================
    function init() {
        if (document.getElementById('robot-pet-v2')) {
            console.log('机器人桌宠V2已存在');
            return;
        }
        
        // 移除旧版本
        const oldRobot = document.getElementById('robot-pet');
        if (oldRobot) {
            oldRobot.remove();
        }
        
        createStyles();
        createHTML();
        bindEvents();
        startAutoBehavior();
        
        setTimeout(() => {
            performAction('waving');
            randomDialog('greeting');
        }, 500);
        
        console.log('🤖 灵动机器人桌宠V2已加载！');
    }
    
    // ==================== 全局接口 ====================
    window.RobotPet = {
        say: showDialog,
        setEmotion: setEmotion,
        performAction: performAction,
        move: randomMove
    };
    
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    
})();

