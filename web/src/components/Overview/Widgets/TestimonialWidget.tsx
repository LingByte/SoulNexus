import React from 'react'
import { WidgetConfig } from '@/types/overview'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { Quote } from 'lucide-react'

interface Testimonial {
  name: string
  role?: string
  company?: string
  content: string
  avatar?: string
  rating?: number
}

interface TestimonialWidgetProps {
  config: WidgetConfig
  data?: Testimonial[]
}

const TestimonialWidget: React.FC<TestimonialWidgetProps> = ({ config, data }) => {
  const { title, props, style } = config
  
  // 确保 testimonials 是数组
  let testimonials: Testimonial[] = []
  if (Array.isArray(data)) {
    testimonials = data
  } else if (Array.isArray(props?.testimonials)) {
    testimonials = props.testimonials
  } else if (data && typeof data === 'object' && !Array.isArray(data)) {
    // 如果 data 是对象，尝试提取数组
    testimonials = data.testimonials || data.items || []
  } else {
    testimonials = [
      {
        name: '客户A',
        role: 'CEO',
        company: '公司A',
        content: '这是一个非常好的产品，帮助我们提高了效率。',
        rating: 5
      },
      {
        name: '客户B',
        role: 'CTO',
        company: '公司B',
        content: '服务非常专业，值得推荐。',
        rating: 5
      },
    ]
  }

  // 确保是数组
  if (!Array.isArray(testimonials)) {
    testimonials = []
  }

  const displayCount = props?.displayCount || 1
  const testimonialsToShow = testimonials.slice(0, displayCount)

  return (
    <Card className="h-full" style={style}>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="h-[calc(100%-60px)] overflow-y-auto">
        <div className="space-y-4">
          {testimonialsToShow.map((testimonial, index) => (
            <div key={index} className="p-4 border rounded-lg bg-muted/30">
              <Quote className="w-6 h-6 text-muted-foreground mb-2 opacity-50" />
              <p className="text-sm mb-4 leading-relaxed">{testimonial.content}</p>
              <div className="flex items-center gap-3">
                {testimonial.avatar ? (
                  <img
                    src={testimonial.avatar}
                    alt={testimonial.name}
                    className="w-10 h-10 rounded-full object-cover"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.display = 'none'
                    }}
                  />
                ) : (
                  <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center text-primary font-semibold">
                    {testimonial.name.charAt(0).toUpperCase()}
                  </div>
                )}
                <div className="flex-1">
                  <div className="font-medium text-sm">{testimonial.name}</div>
                  {(testimonial.role || testimonial.company) && (
                    <div className="text-xs text-muted-foreground">
                      {testimonial.role && testimonial.role}
                      {testimonial.role && testimonial.company && ' · '}
                      {testimonial.company}
                    </div>
                  )}
                  {testimonial.rating && (
                    <div className="flex gap-1 mt-1">
                      {Array.from({ length: 5 }).map((_, i) => (
                        <span
                          key={i}
                          className={`text-xs ${
                            i < testimonial.rating! ? 'text-yellow-400' : 'text-muted-foreground/30'
                          }`}
                        >
                          ★
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export default TestimonialWidget

