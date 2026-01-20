// 灵动机器人桌宠
// 一个真正会动、会走、会说话的桌宠机器人

(function() {
    'use strict';
    
    // ==================== 配置 ====================
    const CONFIG = {
        robotName: 'Robo',
        robotSize: 120,
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
        currentAction: 'idle', // idle, walking, jumping, waving, thinking
        currentEmotion: 'happy', // happy, sad, thinking, surprised, excited
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
            .robot-pet {
                position: fixed;
                width: ${CONFIG.robotSize}px;
                height: ${CONFIG.robotSize}px;
                z-index: 999999;
                cursor: pointer;
                user-select: none;
                transition: transform 0.3s ease;
            }
            
            .robot-pet.dragging {
                cursor: grabbing;
                transform: scale(1.1);
            }
            
            .robot-pet.flipped {
                transform: scaleX(-1);
            }
            
            .robot-pet.flipped.dragging {
                transform: scaleX(-1) scale(1.1);
            }
            
            /* 机器人身体 */
            .robot-body {
                position: absolute;
                width: 100%;
                height: 100%;
                display: flex;
                flex-direction: column;
                align-items: center;
                filter: drop-shadow(0 8px 16px rgba(0, 0, 0, 0.2));
            }
            
            /* 头部 */
            .robot-head {
                width: 60px;
                height: 60px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                border-radius: 50% 50% 45% 45%;
                position: relative;
                animation: robot-head-bob 2s ease-in-out infinite;
                box-shadow: inset 0 -5px 10px rgba(0, 0, 0, 0.2);
            }
            
            /* 天线 */
            .robot-antenna {
                position: absolute;
                top: -15px;
                left: 50%;
                transform: translateX(-50%);
                width: 3px;
                height: 15px;
                background: #667eea;
                animation: robot-antenna-wave 3s ease-in-out infinite;
            }
            
            .robot-antenna::after {
                content: '';
                position: absolute;
                top: -6px;
                left: 50%;
                transform: translateX(-50%);
                width: 8px;
                height: 8px;
                background: #f59e0b;
                border-radius: 50%;
                box-shadow: 0 0 10px #f59e0b;
                animation: robot-antenna-light 1s ease-in-out infinite;
            }
            
            /* 眼睛容器 */
            .robot-eyes {
                position: absolute;
                top: 18px;
                width: 100%;
                display: flex;
                justify-content: center;
                gap: 15px;
            }
            
            .robot-eye {
                width: 12px;
                height: 12px;
                background: white;
                border-radius: 50%;
                position: relative;
                overflow: hidden;
            }
            
            .robot-pupil {
                position: absolute;
                width: 6px;
                height: 6px;
                background: #1f2937;
                border-radius: 50%;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                transition: all 0.2s ease;
            }
            
            /* 嘴巴 */
            .robot-mouth {
                position: absolute;
                bottom: 12px;
                left: 50%;
                transform: translateX(-50%);
                width: 24px;
                height: 12px;
                border: 3px solid white;
                border-top: none;
                border-radius: 0 0 12px 12px;
                transition: all 0.3s ease;
            }
            
            .robot-mouth.happy {
                border-radius: 0 0 12px 12px;
            }
            
            .robot-mouth.sad {
                border-radius: 12px 12px 0 0;
                border-top: 3px solid white;
                border-bottom: none;
            }
            
            .robot-mouth.surprised {
                border-radius: 50%;
                border: 3px solid white;
                width: 16px;
                height: 16px;
            }
            
            /* 身体躯干 */
            .robot-torso {
                width: 50px;
                height: 40px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                border-radius: 8px;
                margin-top: 5px;
                position: relative;
                box-shadow: inset 0 -3px 8px rgba(0, 0, 0, 0.2);
            }
            
            .robot-torso::before {
                content: '';
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                width: 20px;
                height: 20px;
                background: rgba(255, 255, 255, 0.2);
                border-radius: 50%;
                border: 2px solid rgba(255, 255, 255, 0.4);
            }
            
            /* 手臂 */
            .robot-arms {
                position: absolute;
                top: 5px;
                width: 100%;
                display: flex;
                justify-content: space-between;
            }
            
            .robot-arm {
                width: 8px;
                height: 30px;
                background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
                border-radius: 4px;
                animation: robot-arm-swing 1.5s ease-in-out infinite;
            }
            
            .robot-arm.left {
                margin-left: -10px;
            }
            
            .robot-arm.right {
                margin-right: -10px;
                animation-delay: 0.75s;
            }
            
            .robot-arm.waving {
                animation: robot-arm-wave 0.6s ease-in-out infinite;
            }
            
            /* 腿 */
            .robot-legs {
                display: flex;
                gap: 8px;
                margin-top: 3px;
            }
            
            .robot-leg {
                width: 12px;
                height: 20px;
                background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
                border-radius: 6px;
                position: relative;
            }
            
            .robot-foot {
                position: absolute;
                bottom: -5px;
                left: -3px;
                width: 18px;
                height: 8px;
                background: #764ba2;
                border-radius: 4px;
            }
            
            /* 对话气泡 */
            .robot-dialog {
                position: absolute;
                bottom: 100%;
                left: 50%;
                transform: translateX(-50%) translateY(-10px);
                background: white;
                color: #1f2937;
                padding: 12px 16px;
                border-radius: 18px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
                white-space: nowrap;
                font-size: 14px;
                font-weight: 600;
                opacity: 0;
                pointer-events: none;
                transition: all 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55);
                margin-bottom: 15px;
                max-width: 200px;
                white-space: normal;
                text-align: center;
            }
            
            .robot-dialog.show {
                opacity: 1;
                transform: translateX(-50%) translateY(0);
            }
            
            .robot-dialog::after {
                content: '';
                position: absolute;
                top: 100%;
                left: 50%;
                transform: translateX(-50%);
                width: 0;
                height: 0;
                border-left: 8px solid transparent;
                border-right: 8px solid transparent;
                border-top: 8px solid white;
            }
            
            /* 粒子效果 */
            .robot-particle {
                position: absolute;
                width: 6px;
                height: 6px;
                background: #667eea;
                border-radius: 50%;
                pointer-events: none;
                animation: robot-particle-fade 1s ease-out forwards;
            }
            
            /* 表情特效 */
            .robot-emoji {
                position: absolute;
                font-size: 24px;
                pointer-events: none;
                animation: robot-emoji-float 2s ease-out forwards;
            }
            
            /* 动画定义 */
            @keyframes robot-head-bob {
                0%, 100% { transform: translateY(0); }
                50% { transform: translateY(-3px); }
            }
            
            @keyframes robot-antenna-wave {
                0%, 100% { transform: translateX(-50%) rotate(0deg); }
                25% { transform: translateX(-50%) rotate(-10deg); }
                75% { transform: translateX(-50%) rotate(10deg); }
            }
            
            @keyframes robot-antenna-light {
                0%, 100% { opacity: 1; }
                50% { opacity: 0.5; }
            }
            
            @keyframes robot-arm-swing {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-15deg); }
            }
            
            @keyframes robot-arm-wave {
                0%, 100% { transform: rotate(-30deg); }
                50% { transform: rotate(30deg); }
            }
            
            @keyframes robot-particle-fade {
                0% {
                    transform: translateY(0) scale(1);
                    opacity: 1;
                }
                100% {
                    transform: translateY(-30px) scale(0);
                    opacity: 0;
                }
            }
            
            @keyframes robot-emoji-float {
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
            
            /* 走路动画 */
            .robot-pet.walking .robot-leg:nth-child(1) {
                animation: robot-leg-walk-left 0.6s ease-in-out infinite;
            }
            
            .robot-pet.walking .robot-leg:nth-child(2) {
                animation: robot-leg-walk-right 0.6s ease-in-out infinite;
            }
            
            @keyframes robot-leg-walk-left {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(-20deg); }
            }
            
            @keyframes robot-leg-walk-right {
                0%, 100% { transform: rotate(0deg); }
                50% { transform: rotate(20deg); }
            }
            
            /* 跳跃动画 */
            .robot-pet.jumping {
                animation: robot-jump 0.8s ease-in-out;
            }
            
            @keyframes robot-jump {
                0%, 100% { transform: translateY(0); }
                50% { transform: translateY(-60px); }
            }
            
            /* 思考动画 */
            .robot-pet.thinking .robot-head {
                animation: robot-head-think 1s ease-in-out infinite;
            }
            
            @keyframes robot-head-think {
                0%, 100% { transform: rotate(0deg); }
                25% { transform: rotate(-5deg); }
                75% { transform: rotate(5deg); }
            }
        `;
        document.head.appendChild(style);
    }
    
    // ==================== HTML结构 ====================
    function createHTML() {
        const robot = document.createElement('div');
        robot.className = 'robot-pet';
        robot.id = 'robot-pet';
        robot.style.left = state.x + 'px';
        robot.style.top = state.y + 'px';
        
        robot.innerHTML = `
            <div class="robot-body">
                <!-- 对话气泡 -->
                <div class="robot-dialog" id="robot-dialog"></div>
                
                <!-- 头部 -->
                <div class="robot-head">
                    <div class="robot-antenna"></div>
                    <div class="robot-eyes">
                        <div class="robot-eye">
                            <div class="robot-pupil" id="robot-pupil-left"></div>
                        </div>
                        <div class="robot-eye">
                            <div class="robot-pupil" id="robot-pupil-right"></div>
                        </div>
                    </div>
                    <div class="robot-mouth happy" id="robot-mouth"></div>
                </div>
                
                <!-- 躯干 -->
                <div class="robot-torso">
                    <div class="robot-arms">
                        <div class="robot-arm left"></div>
                        <div class="robot-arm right"></div>
                    </div>
                </div>
                
                <!-- 腿 -->
                <div class="robot-legs">
                    <div class="robot-leg">
                        <div class="robot-foot"></div>
                    </div>
                    <div class="robot-leg">
                        <div class="robot-foot"></div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(robot);
    }
    
    // ==================== 功能函数 ====================
    
    // 显示对话
    function showDialog(text) {
        const dialog = document.getElementById('robot-dialog');
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
        const mouth = document.getElementById('robot-mouth');
        if (!mouth) return;
        
        mouth.className = `robot-mouth ${emotion}`;
    }
    
    // 创建粒子
    function createParticle(x, y) {
        const robot = document.getElementById('robot-pet');
        if (!robot) return;
        
        const particle = document.createElement('div');
        particle.className = 'robot-particle';
        particle.style.left = x + 'px';
        particle.style.bottom = y + 'px';
        robot.appendChild(particle);
        
        setTimeout(() => particle.remove(), 1000);
    }
    
    // 创建表情符号
    function createEmoji(emoji) {
        const robot = document.getElementById('robot-pet');
        if (!robot) return;
        
        const emojiEl = document.createElement('div');
        emojiEl.className = 'robot-emoji';
        emojiEl.textContent = emoji;
        emojiEl.style.left = '50%';
        emojiEl.style.top = '-20px';
        emojiEl.style.transform = 'translateX(-50%)';
        robot.appendChild(emojiEl);
        
        setTimeout(() => emojiEl.remove(), 2000);
    }
    
    // 执行动作
    function performAction(action) {
        const robot = document.getElementById('robot-pet');
        if (!robot) return;
        
        // 移除之前的动作类
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
                setEmotion('excited');
                setTimeout(() => {
                    robot.classList.remove('jumping');
                    setEmotion('happy');
                }, 800);
                break;
                
            case 'waving':
                const arms = robot.querySelectorAll('.robot-arm');
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
        const robot = document.getElementById('robot-pet');
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
        
        // 设置朝向
        if (deltaX > 0) {
            state.facingRight = true;
            robot.classList.remove('flipped');
        } else {
            state.facingRight = false;
            robot.classList.add('flipped');
        }
        
        // 移动
        state.velocityX = (deltaX / distance) * CONFIG.moveSpeed;
        state.velocityY = (deltaY / distance) * CONFIG.moveSpeed;
        
        state.x += state.velocityX;
        state.y += state.velocityY;
        
        robot.style.left = state.x + 'px';
        robot.style.top = state.y + 'px';
        
        // 创建脚步粒子
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
        const robot = document.getElementById('robot-pet');
        if (!robot) return;
        
        const robotRect = robot.getBoundingClientRect();
        const robotCenterX = robotRect.left + robotRect.width / 2;
        const robotCenterY = robotRect.top + 30;
        
        const angle = Math.atan2(mouseY - robotCenterY, mouseX - robotCenterX);
        const distance = Math.min(3, Math.sqrt(
            Math.pow(mouseX - robotCenterX, 2) + 
            Math.pow(mouseY - robotCenterY, 2)
        ) / 50);
        
        const eyeX = Math.cos(angle) * distance;
        const eyeY = Math.sin(angle) * distance;
        
        const pupils = robot.querySelectorAll('.robot-pupil');
        pupils.forEach(pupil => {
            pupil.style.transform = `translate(calc(-50% + ${eyeX}px), calc(-50% + ${eyeY}px))`;
        });
    }
    
    // ==================== 事件处理 ====================
    function bindEvents() {
        const robot = document.getElementById('robot-pet');
        if (!robot) return;
        
        // 拖拽
        let dragStartX, dragStartY;
        
        robot.addEventListener('mousedown', (e) => {
            state.isDragging = true;
            robot.classList.add('dragging');
            dragStartX = e.clientX - state.x;
            dragStartY = e.clientY - state.y;
            e.preventDefault();
        });
        
        document.addEventListener('mousemove', (e) => {
            // 眼睛跟随
            updateEyes(e.clientX, e.clientY);
            
            // 拖拽
            if (state.isDragging) {
                state.x = e.clientX - dragStartX;
                state.y = e.clientY - dragStartY;
                
                // 边界检查
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
        
        // 点击
        robot.addEventListener('click', (e) => {
            if (!state.isDragging) {
                const actions = ['jumping', 'waving', 'thinking'];
                const randomAction = actions[Math.floor(Math.random() * actions.length)];
                performAction(randomAction);
                randomDialog('clicked');
            }
        });
        
        // 双击
        robot.addEventListener('dblclick', () => {
            randomMove();
        });
    }
    
    // ==================== 自动行为 ====================
    function startAutoBehavior() {
        // 定期随机对话
        setInterval(() => {
            if (!state.dialogVisible && Math.random() < 0.5) {
                randomDialog('idle');
            }
        }, CONFIG.idleDialogInterval);
        
        // 定期随机移动
        setInterval(() => {
            if (Math.random() < 0.3) {
                randomMove();
            }
        }, CONFIG.autoMoveInterval);
        
        // 定期随机动作
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
        if (document.getElementById('robot-pet')) {
            console.log('机器人桌宠已存在');
            return;
        }
        
        createStyles();
        createHTML();
        bindEvents();
        startAutoBehavior();
        
        // 欢迎语
        setTimeout(() => {
            performAction('waving');
            randomDialog('greeting');
        }, 500);
        
        console.log('🤖 灵动机器人桌宠已加载！');
        console.log('💡 提示: 点击机器人触发动作，双击让它移动，拖拽改变位置');
    }
    
    // ==================== 全局接口 ====================
    window.RobotPet = {
        say: showDialog,
        setEmotion: setEmotion,
        performAction: performAction,
        move: randomMove
    };
    
    // 自动初始化
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
    
})();

