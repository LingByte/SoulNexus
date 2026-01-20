/** admin widgets
 * How to extend:
 *  1. add a new Widget
 *  2. set field.attribute.widget with the new Widget name
 *  3. add the new Widget to the window.AdminWidgets
 * */
const Icons = {
    Yes: `<span class="text-green-600"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
</svg></span>`,
    No: `<span class="text-red-600"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
<path stroke-linecap="round" stroke-linejoin="round" d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
</svg></span>`,
    Unknown: `<span class="text-gray-600"><svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
<path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
</svg></span>`
}

class BaseWidget {
    render(elm) {
        if (this.field.value) {
            this.renderWith(elm, this.field.value || '')
        }
    }

    renderWith(elm, text) {
        if (text && text.length > 40) {
            elm.classList.add('w-72', 'truncate')
        }
        elm.innerText = text
    }
    renderEditLabel(elm) {
        let node = document.createElement('div')
        node.className = 'flex items-center space-x-1'
        elm.appendChild(node)

        if (this.field.required) {
            let r = document.createElement('span')
            r.innerText = '*'
            r.className = 'text-red-600'
            node.appendChild(r)
        }
        let label = document.createElement('span')
        label.innerText = this.field.label
        label.className = 'text-gray-700'
        node.appendChild(label)

        if (this.field.attribute && this.field.attribute.help) {
            let help = document.createElement('span')
            help.innerText = this.field.attribute.help
            help.className = 'text-gray-400 text-xs'
            node.appendChild(help)
        }
    }

    renderEdit(elm) {
        let node = document.createElement('input')
        node.type = 'text'
        node.className = 'block rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 w-full'
        node.value = this.field.value
        node.placeholder = this.field.placeholder || ''
        node.autocomplete = 'off'
        elm.appendChild(node)

        node.addEventListener('change', (e) => {
            e.preventDefault()
            this.field.value = e.target.value
            this.field.dirty = true
        })
    }
}

class BooleanWidget extends BaseWidget {
    render(elm) {
        if (this.field.value === true) {
            elm.innerHTML = Icons.Yes
        } else if (this.field.value === false) {
            elm.innerHTML = Icons.No
        } else {
            elm.innerHTML = Icons.Unknown
        }
    }
    renderEditLabel(elm) {
        let node = document.createElement('input')
        node.type = 'checkbox'
        node.className = 'mr-2 h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600'
        node.checked = this.field.value === true
        node.addEventListener('change', (e) => {
            e.preventDefault()
            this.field.value = e.target.checked
            this.field.dirty = true
        })
        elm.appendChild(node)
        elm.className = 'mt-2 flex items-center'
        super.renderEditLabel(elm)
    }
    renderEdit(elm) {
        // ignore
    }
}
class TextareaWidget extends BaseWidget {
    renderEdit(elm) {
        let node = document.createElement('textarea')
        node.rows = this.field.textareaRows || 3
        node.className = 'block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6'
        node.value = this.field.value
        node.placeholder = this.field.placeholder || ''
        node.addEventListener('change', (e) => {
            e.preventDefault()
            this.field.value = e.target.value
            this.field.dirty = true
        })
        elm.appendChild(node)
    }
}

class DateTimeWidget extends BaseWidget {
    render(elm) {
        if (!this.field.value || this.field.value.Valid === false) {
            return
        }
        if (this.field.value.Valid && this.field.value.Time) {  // golang sql.NullTime
            this.renderWith(elm, new Date(this.field.value.Time).toLocaleString())
        } else {
            this.renderWith(elm, new Date(this.field.value).toLocaleString())
        }
    }
    renderEdit(elm) {
        let node = document.createElement('input')
        node.type = 'datetime-local'
        node.className = 'block rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6'
        let v = this.field.value
        if (typeof v == 'string' && v.length >= 16) {
            node.value = v.substring(0, 16)
        } else if (v && v.Valid && v.Time) {  // golang sql.NullTime
            node.value = v.Time.substring(0, 16)
            this.field.value = v.Time
        }
        node.addEventListener('change', (e) => {
            e.preventDefault()
            let v = e.target.value
            if (v.length == 16) {
                v += ':00Z'
            }
            this.field.value = new Date(v).toISOString()
            this.field.dirty = true
        })
        elm.appendChild(node)
    }
}

class StructWidget extends BaseWidget {
    render(elm) {
        if (this.field.value) {
            this.renderWith(elm, JSON.stringify(this.field.value))
        }
    }
    renderEdit(elm) {
        let node = document.createElement('textarea')
        node.rows = this.field.textareaRows || 5
        node.className = 'block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6'
        let v = this.field.value
        if (v) {
            node.value = JSON.stringify(v, null, 2)
        }
        node.addEventListener('change', (e) => {
            e.preventDefault()
            node.classList.remove('ring-red-600', 'ring-2')
            let v = e.target.value
            try {
                v = JSON.parse(v || '{}')
            } catch (e) {
                node.classList.add('ring-red-600', 'ring-2')
                return
            }
            this.field.value = v
            this.field.dirty = true
        })
        node.placeholder = this.field.placeholder || ''
        elm.appendChild(node)
    }
}

async function loadForeignValues(path) {
    let req = await fetch(path, {
        method: 'POST',
        body: JSON.stringify({
            foreign: true
        }),
    })
    let data = await req.json()
    if (!data.items) {
        return
    }
    return data.items
}

class ForeignKeyWidget extends BaseWidget {
    render(elm) {
        if (this.field.value) {
            this.renderWith(elm, this.field.value.label || this.field.value.value || '')
        }
    }

    renderEdit(elm) {
        let node = document.createElement('select')
        node.className = 'block rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset  focus:ring-indigo-600 sm:text-sm sm:leading-6'
        let firstOption = document.createElement('option')
        firstOption.value = ''
        firstOption.disabled = true
        firstOption.innerText = this.field.placeholder || 'Select A Value'
        node.appendChild(firstOption)

        loadForeignValues(this.field.foreign.path).then(items => {
            if (!items) return

            if (!this.field.value) {
                this.field.value = items[0].value
                this.field.dirty = true
            }

            items.forEach(item => {
                let option = document.createElement('option')
                option.value = item.value
                option.innerText = item.label || item.value
                if (item.value == this.field.value) {
                    option.selected = true
                }
                node.appendChild(option)
            })
        })
        node.addEventListener('change', (e) => {
            e.preventDefault()
            this.field.value = e.target.value
            this.field.dirty = true
        })
        elm.appendChild(node)
    }
}

class SelectWidget extends BaseWidget {
    render(elm) {
        if (this.field.value && this.field.attribute) {
            if (this.field.attribute.choices) {
                let opt = this.field.attribute.choices.find(opt => opt.value == this.field.value)
                if (opt) {
                    this.renderWith(elm, opt.label || opt.value)
                    return
                }
            }
            this.renderWith(elm, this.field.value)
        }
    }

    renderEdit(elm) {
        let node = document.createElement('select')
        node.className = 'block rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset  focus:ring-indigo-600 sm:text-sm sm:leading-6'
        let firstOption = document.createElement('option')
        firstOption.value = ''
        firstOption.disabled = true
        firstOption.innerText = this.field.placeholder || 'Select A Option'
        node.appendChild(firstOption)
        if (this.field.attribute && this.field.attribute.choices) {
            for (let opt of this.field.attribute.choices) {
                let option = document.createElement('option')
                option.value = opt.value
                option.innerText = opt.label || opt.value
                if (opt.value == this.field.value) {
                    option.selected = true
                }
                node.appendChild(option)
            }
        }
        node.addEventListener('change', (e) => {
            e.preventDefault()
            this.field.value = e.target.value
            this.field.dirty = true
        })
        elm.appendChild(node)
    }
}

class PasswordWidget extends BaseWidget {
    render(elm) {
        if (this.field.value) {
            this.renderWith(elm, '********')
        }
    }

    renderEdit(elm) {
        let node = document.createElement('div')
        node.className = 'flex space-x-2'
        let btn = document.createElement('button')
        btn.className = 'inline-flex items-center px-2.5 py-2 border border-transparent text-xs font-medium rounded text-indigo-700 bg-indigo-100 hover:bg-indigo-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500'
        btn.innerText = 'Show Password Form'

        let input = document.createElement('input')
        input.type = 'text'
        input.autocomplete = 'off'
        input.className = 'hidden block w-64 rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6'
        input.placeholder = this.field.placeholder || 'Type password to change'
        input.autocomplete = 'off'
        btn.addEventListener('click', (e) => {
            e.preventDefault()
            input.classList.toggle('hidden')
            if (input.classList.contains('hidden')) {
                this.field.value = this.field.oldValue // don't change the value when hiding
                this.field.dirty = false
                btn.innerText = 'Show Password Form'
            } else {
                btn.innerText = 'Hide Password Form'
            }
        })

        input.addEventListener('change', (e) => {
            e.preventDefault()
            this.field.value = e.target.value
            this.field.dirty = true
        })
        node.appendChild(btn)
        node.appendChild(input)
        elm.appendChild(node)
    }
}

class SelectFilterWidget {
    render(elm) {
        let options = [{ label: 'Empty value', value: null }]
        options.push(...this.field.attribute.choices)

        let singleChoice = false
        if (this.field.attribute && this.field.attribute.singleChoice !== undefined) {
            singleChoice = this.field.attribute.singleChoice
        }
        this.renderWithOptions(elm, options, !singleChoice)
    }

    renderWithOptions(elm, options, multiple = false) {
        let node = document.createElement('div')
        if (options.length > 20) {
            node.className = 'grid grid-cols-3 gap-2'
        } else if (options.length > 10) {
            node.className = 'grid grid-cols-2 gap-2'
        } else {
            node.className = 'grid grid-cols-1 gap-2'
        }

        options.forEach(opt => {
            let option = document.createElement('label')
            option.className = 'flex items-center hover:bg-gray-50 rounded py-2 px-2'
            let input = document.createElement('input')
            input.type = multiple ? 'checkbox' : 'radio'
            input.data = opt
            input.className = `h-4 w-4 ${multiple ? 'rounded' : ''} border-gray-300 text-indigo-600 focus:ring-indigo-500`

            input.addEventListener('change', (e) => {
                e.preventDefault()

                if (!multiple) {
                    node.querySelectorAll('input').forEach(el => {
                        el.checked = el.data.value == e.target.data.value
                    })
                }

                let vals = []
                node.querySelectorAll('input:checked').forEach(n => {
                    vals.push(n.data)
                })
                let selected = null
                if (vals.length > 0) {
                    const op = opt.op || this.op || '='
                    selected = {
                        name: this.name || this.field.name,
                        op: vals.length > 1 ? 'in' : op,
                        value: vals.length > 1 ? vals.map(v => v.value) : vals[0].value,
                        showOp: vals.length > 1 ? 'in' : 'is',
                        showValue: vals.map(v => v.label).join(', '),
                    }
                }
                this.field.onSelect(this.field, selected)
            })
            option.appendChild(input)
            let span = document.createElement('span')
            span.className = 'ml-3 text-sm text-gray-500'
            span.innerText = opt.label
            option.appendChild(span)
            node.appendChild(option)
        })

        let clean = document.createElement('button')
        clean.className = 'mt-4 inline-flex items-center  justify-center px-2.5 py-2 border border-transparent text-xs font-medium rounded text-indigo-700 bg-indigo-100 hover:bg-indigo-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500'
        clean.innerText = 'Clean'
        clean.addEventListener('click', (e) => {
            e.preventDefault()
            node.querySelectorAll('input').forEach(el => {
                el.checked = false
            })
            this.field.onSelect(this.field, null)
        })
        node.appendChild(clean)
        elm.appendChild(node)
    }
}

class BaseFilterWidget {
    render(elm) {

    }
}
class NumberFilterWidget {
    render(elm) {

    }
}
class BooleanFilterWidget extends SelectFilterWidget {
    render(elm) {
        const options = [{ label: 'Empty value', value: null },
        { label: 'Yes', value: true },
        { label: 'No', value: false }]
        super.renderWithOptions(elm, options, true)
    }
}

class DateTimeFilterWidget extends SelectFilterWidget {
    render(elm) {
        let today = new Date()
        today.setHours(0, 0, 0, 0)
        let endOfToday = new Date(today.getTime() + 24 * 60 * 60 * 1000)
        endOfToday.setHours(0, 0, 0, 0)

        let past7 = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000)
        past7.setHours(0, 0, 0, 0)

        let endOfPast7 = new Date(today.getTime() + 24 * 60 * 60 * 1000)
        endOfPast7.setHours(0, 0, 0, 0)

        let thisWeek = new Date(today.getTime() - today.getDay() * 24 * 60 * 60 * 1000)
        thisWeek.setHours(0, 0, 0, 0)
        let endOfThisWeek = new Date(thisWeek.getTime() + 7 * 24 * 60 * 60 * 1000)
        endOfThisWeek.setHours(0, 0, 0, 0)

        let thisMonth = new Date(today.getFullYear(), today.getMonth(), 1)
        thisMonth.setHours(0, 0, 0, 0)
        let endOfThisMonth = new Date(today.getFullYear(), today.getMonth() + 1, 1)
        endOfThisMonth.setHours(0, 0, 0, 0)

        let thisYear = new Date(today.getFullYear(), 0, 1)
        thisYear.setHours(0, 0, 0, 0)
        let endOfThisYear = new Date(today.getFullYear() + 1, 0, 1)
        endOfThisYear.setHours(0, 0, 0, 0)

        const options = [
            { label: 'Any date', value: null, op: 'is not' },
            { label: 'Today', value: [today.toISOString(), endOfToday.toISOString()] },
            { label: 'Past 7 days', value: [past7.toISOString(), endOfPast7.toISOString()] },
            { label: 'This week', value: [thisWeek.toISOString(), endOfThisWeek.toISOString()] },
            { label: 'This month', value: [thisMonth.toISOString(), endOfThisMonth.toISOString()] },
            { label: 'This year', value: [thisYear.toISOString(), endOfThisYear.toISOString()] },
        ]
        this.op = 'between'
        super.renderWithOptions(elm, options, false)
    }
}
class ForeignKeyFilterWidget extends SelectFilterWidget {
    render(elm) {
        this.name = this.field.foreign.field // use the foreign field name as the filter name
        let options = []
        if (this.field.canNull) {
            options.push({ label: 'Empty value', value: null })
        }

        loadForeignValues(this.field.foreign.path).then(items => {
            if (items) {
                options.push(...items)
            }

            let singleChoice = false
            if (this.field.attribute && this.field.attribute.singleChoice !== undefined) {
                singleChoice = this.field.attribute.singleChoice
            }

            this.renderWithOptions(elm, options, !singleChoice)
        })
    }
}

window.AdminWidgets = {
    'string': BaseWidget,
    'uint': BaseWidget,
    'int': BaseWidget,
    'float': BaseWidget,

    'bool': BooleanWidget,
    'textarea': TextareaWidget,
    'datetime': DateTimeWidget,
    'foreign': ForeignKeyWidget,
    'struct': StructWidget,
    'password': PasswordWidget,
    'select': SelectWidget,
}

window.AdminFilterWidgets = {
    'string': BaseFilterWidget,
    'uint': NumberFilterWidget,
    'int': NumberFilterWidget,
    'float': NumberFilterWidget,

    'bool': BooleanFilterWidget,
    'datetime': DateTimeFilterWidget,
    'foreign': ForeignKeyFilterWidget,
    'select': SelectFilterWidget,
}


function getWidget(field, col) {
    let widgetType = null
    if (field.foreign) {
        widgetType = 'foreign'
    } else {
        switch (field.type) {
            case 'string':
            case 'bool':
            case 'datetime':
            case 'uint':
            case 'int':
            case 'float':
                widgetType = field.type
                break
            default:
                widgetType = 'struct'
                break
        }
    }

    if (field.attribute) {
        if (field.attribute.widget) {
            widgetType = field.attribute.widget
        } else if (field.attribute.choices) {
            widgetType = 'select'
        }
    }

    if (field.tag && /size:(\d+)/.test(field.tag)) {
        field.size = parseInt(field.tag.match(/size:(\d+)/)[1])
    }

    if (widgetType == 'string' && field.size === undefined) {
        widgetType = 'textarea'
    }

    let widgetCls = window.AdminWidgets[widgetType]
    if (!widgetCls) {
        //
        console.warn(`Widget ${widgetType} not found, using struct widget`)
        widgetCls = window.AdminWidgets['struct']
    }
    let widget = new widgetCls()
    widget.field = field
    widget.col = col
    return widget
}


function getFilterWidget(field) {
    let widgetType = null
    if (field.foreign) {
        widgetType = 'foreign'
    } else {
        switch (field.type) {
            case 'string':
            case 'bool':
            case 'datetime':
            case 'uint':
            case 'int':
            case 'float':
                widgetType = field.type
                break
            default:
                widgetType = 'string'
                break
        }
    }

    if (field.attribute) {
        if (field.attribute.filterWidget) {
            widgetType = field.attribute.filterWidget
        } else if (field.attribute.choices) {
            widgetType = 'select'
        }
    }
    let widgetCls = window.AdminFilterWidgets[widgetType]
    if (!widgetCls) {
        //
        console.warn(`FilterWidget ${widgetType} not found, using string widget`)
        widgetCls = window.AdminFilterWidgets['string']
    }
    let widget = new widgetCls()
    widget.field = field
    return widget
}

document.addEventListener('alpine:init', () => {
    Alpine.directive('admin-render', (el, { expression }, { evaluate }) => {
        let col = evaluate(expression)
        col.field.value = col.value
        getWidget(col.field, col).render(el)
    })

    Alpine.directive('admin-edit-label', (el, { expression }, { evaluate }) => {
        let field = evaluate(expression)
        getWidget(field).renderEditLabel(el)
    })

    Alpine.directive('admin-edit', (el, { expression }, { evaluate }) => {
        let field = evaluate(expression)
        getWidget(field).renderEdit(el)
    })
    Alpine.directive('admin-filter', (el, { expression }, { Alpine, evaluate }) => {
        let field = evaluate(expression)
        getFilterWidget(field).render(el, Alpine)
    })
})