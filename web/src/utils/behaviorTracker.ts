// 行为追踪工具类
export interface MouseTrackPoint {
  x: number;
  y: number;
  timestamp: number;
}

export interface BehaviorData {
  mouseTrack: MouseTrackPoint[];
  formFillTime: number;
  keystrokePattern: string;
  formData: Record<string, string>;
}

export class BehaviorTracker {
  private mouseTrack: MouseTrackPoint[] = [];
  private keystrokeIntervals: number[] = [];
  private lastKeystrokeTime: number = 0;
  private formStartTime: number = 0;
  private isTracking: boolean = false;
  private trackingElement: HTMLElement | null = null;

  constructor() {
    this.handleMouseMove = this.handleMouseMove.bind(this);
    this.handleKeyDown = this.handleKeyDown.bind(this);
  }

  // 开始追踪行为
  startTracking(element?: HTMLElement): void {
    if (this.isTracking) {
      return;
    }

    this.isTracking = true;
    this.formStartTime = Date.now();
    this.mouseTrack = [];
    this.keystrokeIntervals = [];
    this.lastKeystrokeTime = 0;
    this.trackingElement = element || document.body;

    // 添加事件监听器
    this.trackingElement.addEventListener('mousemove', this.handleMouseMove, { passive: true });
    this.trackingElement.addEventListener('keydown', this.handleKeyDown, { passive: true });

    console.log('Behavior tracking started');
  }

  // 停止追踪行为
  stopTracking(): void {
    if (!this.isTracking) {
      return;
    }

    this.isTracking = false;

    // 移除事件监听器
    if (this.trackingElement) {
      this.trackingElement.removeEventListener('mousemove', this.handleMouseMove);
      this.trackingElement.removeEventListener('keydown', this.handleKeyDown);
      this.trackingElement = null;
    }

    console.log('Behavior tracking stopped');
  }

  // 处理鼠标移动事件
  private handleMouseMove(event: MouseEvent): void {
    if (!this.isTracking) {
      return;
    }

    const point: MouseTrackPoint = {
      x: event.clientX,
      y: event.clientY,
      timestamp: Date.now()
    };

    this.mouseTrack.push(point);

    // 限制轨迹点数量，避免内存过度使用
    if (this.mouseTrack.length > 1000) {
      this.mouseTrack = this.mouseTrack.slice(-500); // 保留最后500个点
    }
  }

  // 处理按键事件
  private handleKeyDown(event: KeyboardEvent): void {
    if (!this.isTracking) {
      return;
    }

    const currentTime = Date.now();
    
    if (this.lastKeystrokeTime > 0) {
      const interval = currentTime - this.lastKeystrokeTime;
      this.keystrokeIntervals.push(interval);
    }

    this.lastKeystrokeTime = currentTime;

    // 限制按键间隔数量
    if (this.keystrokeIntervals.length > 200) {
      this.keystrokeIntervals = this.keystrokeIntervals.slice(-100);
    }
  }

  // 获取行为数据
  getBehaviorData(formData: Record<string, string>): BehaviorData {
    const formFillTime = Date.now() - this.formStartTime;
    const keystrokePattern = JSON.stringify(this.keystrokeIntervals);

    return {
      mouseTrack: this.mouseTrack,
      formFillTime,
      keystrokePattern,
      formData
    };
  }

  // 重置追踪数据
  reset(): void {
    this.mouseTrack = [];
    this.keystrokeIntervals = [];
    this.lastKeystrokeTime = 0;
    this.formStartTime = Date.now();
  }

  // 获取追踪统计信息
  getTrackingStats(): {
    mousePoints: number;
    keystrokeCount: number;
    trackingDuration: number;
  } {
    return {
      mousePoints: this.mouseTrack.length,
      keystrokeCount: this.keystrokeIntervals.length,
      trackingDuration: Date.now() - this.formStartTime
    };
  }

  // 检查是否有足够的行为数据
  hasEnoughData(): boolean {
    return this.mouseTrack.length >= 10 && this.keystrokeIntervals.length >= 5;
  }

  // 添加人工延迟（用于测试）
  static addHumanDelay(): Promise<void> {
    const delay = Math.random() * 200 + 100; // 100-300ms随机延迟
    return new Promise(resolve => setTimeout(resolve, delay));
  }

  // 模拟人类鼠标轨迹（用于测试）
  static simulateHumanMouseTrack(startX: number, startY: number, endX: number, endY: number): MouseTrackPoint[] {
    const points: MouseTrackPoint[] = [];
    const steps = Math.floor(Math.random() * 20) + 10; // 10-30个点
    const startTime = Date.now();

    for (let i = 0; i <= steps; i++) {
      const progress = i / steps;
      
      // 添加一些随机性和曲线
      const randomX = (Math.random() - 0.5) * 20;
      const randomY = (Math.random() - 0.5) * 20;
      
      // 使用贝塞尔曲线模拟自然轨迹
      const controlX = (startX + endX) / 2 + (Math.random() - 0.5) * 100;
      const controlY = (startY + endY) / 2 + (Math.random() - 0.5) * 100;
      
      const x = Math.round(
        (1 - progress) * (1 - progress) * startX +
        2 * (1 - progress) * progress * controlX +
        progress * progress * endX + randomX
      );
      
      const y = Math.round(
        (1 - progress) * (1 - progress) * startY +
        2 * (1 - progress) * progress * controlY +
        progress * progress * endY + randomY
      );

      // 添加时间间隔的随机性
      const timeOffset = Math.random() * 50 + 10; // 10-60ms间隔
      
      points.push({
        x,
        y,
        timestamp: startTime + i * timeOffset
      });
    }

    return points;
  }

  // 模拟人类按键模式（用于测试）
  static simulateHumanKeystroke(textLength: number): number[] {
    const intervals: number[] = [];
    
    for (let i = 0; i < textLength - 1; i++) {
      // 模拟人类按键间隔：80-300ms，偶尔有更长的停顿
      let interval: number;
      
      if (Math.random() < 0.1) {
        // 10%概率有较长停顿（思考时间）
        interval = Math.random() * 1000 + 500; // 500-1500ms
      } else {
        // 正常按键间隔
        interval = Math.random() * 220 + 80; // 80-300ms
      }
      
      intervals.push(Math.round(interval));
    }
    
    return intervals;
  }
}

// 全局行为追踪器实例
export const globalBehaviorTracker = new BehaviorTracker();

// 自动在页面加载时开始追踪
if (typeof window !== 'undefined') {
  window.addEventListener('load', () => {
    // 延迟启动，避免影响页面加载性能
    setTimeout(() => {
      globalBehaviorTracker.startTracking();
    }, 1000);
  });

  // 页面卸载时停止追踪
  window.addEventListener('beforeunload', () => {
    globalBehaviorTracker.stopTracking();
  });
}