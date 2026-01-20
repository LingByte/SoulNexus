async function parseResponseError(resp) {
    let text = undefined
    try {
        text = await resp.text()
        let data = JSON.parse(text)
        return data.error || text
    } catch (err) {
        return text || resp.statusText
    }
}

class ConfirmAction {
    constructor() {
        this.reset()
    }
    reset() {
        this.show = false
        this.action = {
            name: '',
            label: '',
            title: '',
            class: '',
            path: '',
            text: '',
            onDone: null,
            onFail: null,
        }
        this.keys = []
    }
    confirm({ action, keys }) {
        this.reset()
        this.action = Object.assign(this.action, action)
        this.keys = keys
        this.show = true
    }
    cancel(event) {
        if (event) {
            event.preventDefault()
        }
        this.show = false
        this.reset()
    }
}
class Toasts {
    constructor() {
        this.reset()
    }

    get class() {
        if (this.pending) {
            return 'bg-violet-50 border border-violet-200 text-sm text-violet-600 rounded-md p-4 w-64'
        }
        if (this.level === 'error') {
            return 'bg-orange-50 border border-orange-200 text-sm text-orange-600 rounded-md p-4'
        } else if (this.level === 'info') {
            return 'bg-blue-50 border border-blue-200 text-sm text-blue-600 rounded-md p-4'
        }
        return ''
    }
    reset() {
        this.show = false
        this.pending = false
        this.text = ''
        this.level = ''
    }
    info(text, timeout = 6000) {
        this.reset()
        this.text = text
        this.level = 'info'
        this.show = true
        setTimeout(() => {
            this.reset()
        }, timeout)
    }
    error(text, timeout = 10000) {
        this.reset()
        this.text = text
        this.level = 'error'
        this.show = true
        setTimeout(() => {
            this.reset()
        }, timeout)
    }
    doing(text) {
        this.reset()
        this.text = text
        this.pending = true
        this.show = true
    }
}

class QueryResult {
    constructor() {
        this.reset()
    }
    reset() {
        this.countPerPage = 20
        this.pos = 0
        this.total = 0
        this.limit = 20
        this.rows = []
        this.count = 0
        this.selected = 0
        this.keyword = ''
        this.orders = []
        this.filters = []
    }

    async attach(data) {
        this.pos = data.pos || 0
        this.total = data.total || 0
        this.limit = data.limit || 20
        let items = data.items || []
        this.count = items.length

        let current = Alpine.store('current')
        items = items.map(item => {
            return {
                primaryValue: current.getPrimaryValue(item),
                selected: false,
                rawData: item,
                cols: current.shows.map(field => {
                    return {
                        value: item[field.name],
                        field,
                        name: field.name,
                        primary: field.primary,
                    }
                }),
                ...item._adminExtra,
            }
        })

        if (current.prepareResult) {
            await current.prepareResult(items, this.total)
        }
        this.rows = items
    }

    get posValue() {
        if (this.count == 0) { return 0 }
        return this.pos + 1
    }

    queryprev(event) {
        if (event) {
            event.preventDefault()
        }
        if (this.pos == 0) {
            return
        }
        this.pos = this.pos - this.countPerPage
        if (this.pos < 0) {
            this.pos = 0
        }
        this.refresh()
    }

    querynext(event) {
        if (event) {
            event.preventDefault()
        }
        let pos = this.pos + this.countPerPage
        if (pos >= this.total) {
            return
        }
        this.pos = pos
        this.refresh()
    }

    selectAll(event) {
        this.rows.forEach(row => {
            row.selected = !row.selected
        })
        this.selected = this.rows.filter(row => row.selected).length
    }

    selectResult(event) {
        event.preventDefault()
        this.rows.forEach(row => {
            row.selected = true
        })
        document.getElementById('btn_selectAll').checked = true
        this.selected = this.total
    }

    onSelectRow(event, row) {
        row.selected = !row.selected
        this.selected = this.rows.filter(row => row.selected).length
    }

    setFilters(filters) {
        this.filters.splice(0, this.filters.length)
        if (!filters) {
            return this
        }

        filters.forEach(f => {
            if (f.isGroup) {
                f.value.forEach(sub => {
                    this.filters.push(sub)
                })
            } else {
                this.filters.push(f)
            }
        })
        return this
    }

    setOrders(orders) {
        this.orders.splice(0, this.orders.length)
        this.orders.push(...orders)
        return this
    }

    toggleOrder(field, value) {
        let of = this.orders.find(o => o.name === field.name)
        if (!of) {
            console.error(`order field ${field.name} not found`, this.orders)
            return
        }

        if (value === '') {
            field.sort = ''
            of.op = ''
            this.refresh()
            return
        }
        if (field.sort == '' || field.sort === 'desc') {
            field.sort = 'asc'
        } else if (field.sort === 'asc') {
            field.sort = 'desc'
        }
        of.op = field.sort
        this.refresh()
    }

    refresh(source) {
        let current = Alpine.store('current')
        let query = {
            keyword: this.keyword,
            pos: this.pos,
            limit: this.countPerPage,
            filters: this.filters,
            orders: this.orders
        }

        if (current.prepareQuery) {
            let q = current.prepareQuery(query, source)
            if (!q) {
                // cancel query
                return
            }
            query = q
        }

        const toasts = Alpine.store('toasts')
        toasts.doing('Loading...')

        fetch(current.path, {
            method: 'POST',
            body: JSON.stringify(query),
        }).then(resp => {
            if (!resp.ok) {
                parseResponseError(resp).then(err => {
                    toasts.error(err)
                })
                return
            }

            resp.json().then(data => {
                this.rows = []
                this.attach(data).then(() => {
                    toasts.reset()
                })
            })
        }).catch(err => {
            toasts.error(err)
        })
    }

    onDeleteOne(event) {
        Alpine.store('confirmAction').confirm({
            action: {
                method: 'DELETE',
                label: 'Delete',
                name: 'Delete',
                path: Alpine.store('current').path,
                class: 'text-white bg-red-500 hover:bg-red-700',
            },
            keys: [Alpine.store('editobj').primaryValue]
        })
    }

    doAction(event) {
        event.preventDefault()
        let { action, keys } = Alpine.store('confirmAction')

        Alpine.store('editobj').closeEdit()
        Alpine.store('confirmAction').cancel()

        Alpine.store('current').doAction(action, keys).then(() => {
            this.rows.forEach(row => {
                row.selected = false
            })
            this.selected = 0
            let btn_selectAll = document.getElementById('btn_selectAll')
            if (btn_selectAll) {
                btn_selectAll.checked = false
            }
            Alpine.store('toasts').info(`${action.name} all records done`)
            this.refresh()
        }).catch(err => {
            Alpine.store('toasts').error(`${action.name} fail : ${err.toString()}`)
        })
    }
}
class EditObject {
    constructor({ mode, title, fields, names, primaryValue, row }) {
        this.mode = mode || undefined
        this.title = title || ''
        this.fields = fields || []
        this.names = names || {}
        this.primaryValue = primaryValue || undefined
        this.row = row || undefined
    }

    get apiUrl() {
        return Alpine.store('current').buildApiUrl(this.row)
    }

    async doSave(ev, closeWhenDone = true) {
        try {
            if (this.mode == 'create') {
                const obj = await Alpine.store('current').doCreate(this.fields)
                this.primaryValue = Alpine.store('current').getPrimaryValue(obj)
            } else {
                await Alpine.store('current').doSave(this.primaryValue, this.fields.filter(f => f.dirty))
            }

            if (closeWhenDone) {
                this.closeEdit(ev)
            } else {
                this.mode = 'edit'
            }
            Alpine.store('queryresult').refresh()
            Alpine.store('toasts').info(`Save Done`)
        } catch (err) {
            console.error(err)
            Alpine.store('toasts').error(`Save Fail: ${err.toString()}`)
            this.closeEdit(ev)
        }
    }
    closeEdit(event, cancel = false) {
        this.mode = undefined
    }
}
class AdminObject {
    constructor(meta) {
        this.permissions = meta.permissions || {}
        this.desc = meta.desc
        this.name = meta.name
        this.path = meta.path
        this.group = meta.group
        this.listpage = meta.listpage || 'list.html'
        this.editpage = meta.editpage || 'edit.html'
        this.primaryKeys = meta.primaryKeys
        this.uniqueKeys = meta.uniqueKeys
        this.pluralName = meta.pluralName
        this.scripts = meta.scripts || []
        this.styles = meta.styles || []
        this.icon = meta.icon
        this.invisible = meta.invisible || false
        let fields = meta.fields || []
        let requireds = meta.requireds || []


        this.fields = fields.map(f => {
            const headerName = f.label || f.name
            f.headerName = headerName.toUpperCase().replace(/_/g, ' ')
            f.primary = f.primary
            f.required = requireds.includes(f.name)

            if (/int/i.test(f.type)) {
                f.type = 'int'
            }

            if (/float/i.test(f.type)) {
                f.type = 'float'
            }

            f.defaultValue = () => {
                if (f.attribute && f.attribute.default !== undefined) {
                    return f.attribute.default
                }
                switch (f.type) {
                    case 'bool': return false
                    case 'int': return 0
                    case 'uint': return 0
                    case 'float': return 0.0
                    case 'datetime': return ''
                    case 'string': return ''
                    default: return null
                }
            }
            // convert value from string to type
            f.unmarshal = (value) => {
                if (value === null || value === undefined) {
                    return value
                }

                if (f.foreign) {
                    return value
                }

                switch (f.type) {
                    case 'bool':
                        if (value === 'true') { return true }
                        return value
                    case 'uint':
                    case 'int': {
                        let v = parseInt(value)
                        if (isNaN(v)) { return undefined }
                        return v
                    }
                    case 'float': {
                        let v = parseFloat(value)
                        if (isNaN(v)) { return undefined }
                        return v
                    }
                    case 'datetime':
                    case 'string':
                        return value
                    default:
                        if (typeof value === 'string') {
                            return JSON.parse(value)
                        }
                        return value
                }
            }
            return f
        })

        let filterFields = (names, defaults) => {
            if (!names) {
                return defaults || []
            }
            return (names || []).map(name => {
                return fields.find(f => f.name === name)
            }).filter(f => f)
        }

        this.shows = filterFields(meta.shows, fields)
        this.editables = filterFields(meta.editables, fields)
        this.searchables = filterFields(meta.searchables)
        this.filterables = filterFields(meta.filterables)
        this.orderables = filterFields(meta.orderables)
        this.orders = meta.orders || []

        this.orderables.forEach(f => {
            const o = this.orders.find(of => of.name === f.name)
            if (!o) {
                this.orders.push({ name: f.name, op: '' })
            }
        })

        this.shows.forEach(f => {
            const o = this.orders.find(of => of.name === f.name)
            f.sort = o ? o.op : ''
            f.canSort = this.orderables.find(of => of.name === f.name) !== undefined
        })

        this.filterables.forEach(f => {
            f.onSelect = this.onFilterSelect.bind(this)
        })

        let actions = meta.actions || []
        // check user can delete
        if (this.permissions.can_delete) {
            actions.push({
                method: 'DELETE',
                name: 'Delete',
                label: 'Delete',
                class: 'text-white bg-red-500 hover:bg-red-700',
            })
        }

        this.actions = actions.filter(action => !action.withoutObject).map(action => {
            let path = this.path
            if (action.path) {
                path = `${path}${action.path}`
            }
            action.path = path
            action.onclick = () => {
                let keys = []
                let queryresult = Alpine.store('queryresult')
                for (let i = 0; i < queryresult.rows.length; i++) {
                    if (queryresult.rows[i].selected) {
                        keys.push(queryresult.rows[i].primaryValue)
                    }
                }
                Alpine.store('confirmAction').confirm({ action, keys })
            }
            if (!action.class) {
                action.class = 'bg-white text-gray-900 ring-1 ring-inset ring-gray-300 hover:bg-gray-50'
            }
            if (!action.label) {
                action.label = action.name
            }
            return action
        })
    }

    onFilterSelect(filter, value) {
        filter.selected = value || {}
        let filters = this.filterables.filter(f => f.selected && f.selected.op).map(f => f.selected)
        Alpine.store('queryresult').setFilters(filters).refresh()
    }

    get hasFilterSelected() {
        return this.filterables.some(f => f.selected && f.selected.op)
    }
    get selectedFilters() {
        return this.filterables.filter(f => f.selected && f.selected.op)
    }

    getPrimaryValue(row) {
        let vals = {}
        let keys = this.primaryKeys || this.uniqueKeys || []
        keys.forEach(key => {
            let f = this.fields.find(f => f.name === key)
            let v = row[key]
            if (v !== undefined) {
                if (f.foreign) {
                    vals[f.foreign.field] = v.value
                } else {
                    vals[key] = v
                }

            }
        })
        return vals
    }

    buildApiUrl(row) {
        if (!row) {
            return ''
        }
        let vals = ['api', this.name.toLowerCase()]
        let keys = this.primaryKeys || this.uniqueKeys || []
        keys.forEach(key => {
            let f = this.fields.find(f => f.name === key)
            let v = row.rawData[key]
            if (v !== undefined) {
                if (f.foreign) {
                    v = v.value
                }
                vals.push(v)
            }
        })
        let config = Alpine.store('config')
        let api_host = config.api_host || location.origin
        if (!api_host.endsWith('/')) {
            api_host += '/'
        }
        return `${api_host}${vals.join('/')}`
    }
    get active() {
        return Alpine.store('current') === this
    }

    get showSearch() {
        return this.searchables.length > 0
    }
    get showFilter() {
        return this.filterables.length > 0
    }

    async doSave(keys, vals) {
        let values = {}
        vals.forEach(v => {
            values[v.name] = v.unmarshal(v.value)
        })
        let params = new URLSearchParams(keys).toString()
        let resp = await fetch(`${this.path}?${params}`, {
            method: 'PATCH',
            body: JSON.stringify(values),
        })
        if (resp.status != 200) {
            throw new Error(await parseResponseError(resp))
        }
        return await resp.json()
    }

    async doCreate(vals) {
        let values = {}
        vals.forEach(v => {
            values[v.name] = v.unmarshal(v.value)
        })

        let resp = await fetch(`${this.path}`, {
            method: 'PUT',
            body: JSON.stringify(values),
        })
        if (resp.status != 200) {
            throw new Error(await parseResponseError(resp))
        }
        return await resp.json()
    }

    async doAction(action, keys) {
        if (action.batch) {
            let items = {
                "keys": JSON.stringify(keys)
            }
            keys = [items]
        }

        for (let i = 0; i < keys.length; i++) {
            Alpine.store('toasts').doing(`${i + 1}/${keys.length}`)
            let params = new URLSearchParams(keys[i]).toString()
            let resp = await fetch(`${action.path}?${params}`, {
                method: action.method || 'POST',
            })
            if (resp.status != 200) {
                let reason = await parseResponseError(resp)
                Alpine.store('toasts').error(`${action.name} fail : ${reason}`)
                if (action.onFail) {
                    let result = await resp.text()
                    action.onFail(keys[i], result)
                }
                break
            }
            if (action.onDone) {
                let result = await resp.json()
                action.onDone(keys[i], result)
            } else {
                // if response is download file
                let contentDisposition = resp.headers.get('content-disposition')
                if (contentDisposition) {
                    let filename = contentDisposition.split('filename=')[1]
                    let blob = await resp.blob()
                    let url = window.URL.createObjectURL(blob)
                    let a = document.createElement('a')
                    a.href = url
                    a.download = filename
                    a.click()
                    window.URL.revokeObjectURL(url)
                }
            }
            Alpine.store('toasts').reset()
        }
    }
}

const adminapp = () => ({
    site: {},
    navmenus: [],
    loadScripts: {},
    loadStyles: {},
    async init() {
        // Initialize all stores with proper default values
        Alpine.store('toasts', new Toasts())
        Alpine.store('queryresult', new QueryResult())
        Alpine.store('current', {
            active: false,
            pluralName: '',
            desc: '',
            name: ''
        })
        Alpine.store('switching', false)
        Alpine.store('loading', true)
        Alpine.store('confirmAction', new ConfirmAction())
        Alpine.store('editobj', new EditObject({}))

        this.$router.config({ mode: 'hash', base: '/api/admin/' })
        let resp = await fetch('./admin.json', {
            method: 'POST',
            cache: "no-store",
        })
        let meta = await resp.json()
        this.site = meta.site
        let objects = meta.objects.map(obj => new AdminObject(obj))
        Alpine.store('objects', objects)
        Alpine.store('config', meta.site)

        if (meta.site.sitename) {
            document.title = `${meta.site.sitename}`
        }
        if (meta.site.slogan) {
            document.title = `${document.title} | ${meta.site.slogan}`
        }

        if (meta.site.favicon_url) {
            let link = document.createElement('link')
            link.rel = 'shortcut icon'
            link.href = meta.site.favicon_url
            document.head.appendChild(link)
        }

        this.user = meta.user
        this.user.name = this.user.firstName || this.user.email
        this.buildNavMenu()
        this.loadSidebar()
        this.loadAllScripts(objects)

        // Make adminApp instance globally accessible for profile page
        window.adminAppInstance = this

        this.$store.loading = false
        this.onLoad()
    },
    
    showProfile(event) {
        if (event) {
            event.preventDefault()
        }
        this.$store.switching = false
        if (this.$store.editobj) {
            this.$store.editobj.mode = undefined
        }
        
        requestAnimationFrame(() => {
            let elm = document.getElementById('query_content')
            if (!elm) {
                setTimeout(() => this.showProfile(), 200)
                return
            }
            
            let user = this.user || {}
            let html = '<div class="space-y-6">'
            
            // Profile Header
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-6">'
            html += '<div class="flex items-center gap-4">'
            if (user.avatar) {
                html += '<img src="' + user.avatar + '" alt="Avatar" class="w-16 h-16 rounded-full border border-gray-200">'
            } else {
                html += '<div class="w-16 h-16 rounded-full border border-gray-200 bg-gray-100 flex items-center justify-center text-xl font-semibold text-gray-600">' + (user.name?.[0] || 'U').toUpperCase() + '</div>'
            }
            html += '<div>'
            html += '<h1 class="text-xl font-semibold text-gray-900 mb-1">' + (user.name || '用户') + '</h1>'
            html += '<p class="text-sm text-gray-500">' + (user.email || '') + '</p>'
            html += '</div>'
            html += '</div>'
            html += '</div>'
            
            // User Info Cards
            html += '<div class="grid grid-cols-1 md:grid-cols-2 gap-4">'
            
            // Basic Info
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5">'
            html += '<h3 class="text-sm font-semibold text-gray-900 mb-3">基本信息</h3>'
            html += '<div class="space-y-2">'
            html += '<div class="flex justify-between py-2 border-b border-gray-100">'
            html += '<span class="text-xs text-gray-600">用户ID</span>'
            html += '<span class="text-xs font-medium text-gray-900">' + (user.id || 'N/A') + '</span>'
            html += '</div>'
            html += '<div class="flex justify-between py-2 border-b border-gray-100">'
            html += '<span class="text-xs text-gray-600">邮箱</span>'
            html += '<span class="text-xs font-medium text-gray-900">' + (user.email || 'N/A') + '</span>'
            html += '</div>'
            html += '<div class="flex justify-between py-2 border-b border-gray-100">'
            html += '<span class="text-xs text-gray-600">角色</span>'
            html += '<span class="text-xs font-medium text-gray-900">' + (user.role || 'Administrator') + '</span>'
            html += '</div>'
            html += '<div class="flex justify-between py-2">'
            html += '<span class="text-xs text-gray-600">状态</span>'
            html += '<span class="text-xs font-medium px-2 py-0.5 rounded bg-gray-100 text-gray-700">' + (user.enabled ? '已启用' : '已禁用') + '</span>'
            html += '</div>'
            html += '</div>'
            html += '</div>'
            
            // Account Stats
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5">'
            html += '<h3 class="text-sm font-semibold text-gray-900 mb-3">账户统计</h3>'
            html += '<div class="grid grid-cols-2 gap-3">'
            html += '<div class="text-center p-3 border border-gray-200 rounded">'
            html += '<div class="text-base font-semibold text-gray-900 mb-1">' + (user.createdAt ? new Date(user.createdAt).toLocaleDateString('zh-CN') : 'N/A') + '</div>'
            html += '<div class="text-xs text-gray-500">注册日期</div>'
            html += '</div>'
            html += '<div class="text-center p-3 border border-gray-200 rounded">'
            html += '<div class="text-base font-semibold text-gray-900 mb-1">' + (user.lastLogin ? new Date(user.lastLogin).toLocaleDateString('zh-CN') : '从未') + '</div>'
            html += '<div class="text-xs text-gray-500">最后登录</div>'
            html += '</div>'
            html += '</div>'
            html += '</div>'
            
            html += '</div>'
            
            // Actions
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5">'
            html += '<h3 class="text-sm font-semibold text-gray-900 mb-3">账户操作</h3>'
            html += '<div class="flex flex-wrap gap-2">'
            html += '<button onclick="window.adminAppInstance.showChangePassword()" class="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-50 border border-gray-300 rounded hover:bg-gray-100 transition-colors">修改密码</button>'
            html += '<button onclick="window.adminAppInstance.showEditProfile()" class="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-50 border border-gray-300 rounded hover:bg-gray-100 transition-colors">编辑资料</button>'
            html += '<a href="/api/auth/logout?next=/api/auth/login" class="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-50 border border-gray-300 rounded hover:bg-gray-100 transition-colors">退出登录</a>'
            html += '</div>'
            html += '</div>'
            
            html += '</div>'
            
            elm.innerHTML = html
            elm.style.display = 'block'
            Alpine.initTree(elm)
            
            // Make instance globally accessible
            window.adminAppInstance = this
        })
    },

    async loadSystemStatus() {
        try {
            let resp = await fetch('/api/system/status', {
                method: 'GET',
                cache: "no-store",
            })
            if (!resp.ok) {
                throw new Error('Failed to fetch system status')
            }
            let data = await resp.json()
            let statusList = document.getElementById('system_status_list')
            if (!statusList) {
                setTimeout(() => this.loadSystemStatus(), 500)
                return
            }
            
            if ((data.success || data.code === 200) && data.data) {
                let status = data.data
                let statusMap = {
                    'database': '数据库',
                    'cache': '缓存服务',
                    'api': 'API服务',
                    'storage': '存储服务'
                }
                let html = ''
                Object.keys(statusMap).forEach(key => {
                    let isOnline = status[key] === true
                    html += '<div class="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">'
                    html += '<div class="flex items-center gap-2">'
                    html += '<div class="w-1.5 h-1.5 rounded-full ' + (isOnline ? 'bg-green-500' : 'bg-red-500') + '"></div>'
                    html += '<span class="text-xs text-gray-700">' + statusMap[key] + '</span>'
                    html += '</div>'
                    html += '<div class="flex items-center gap-3">'
                    html += '<span class="text-xs font-medium ' + (isOnline ? 'text-green-600' : 'text-red-600') + '">' + (isOnline ? '通' : '不通') + '</span>'
                    html += '</div>'
                    html += '</div>'
                })
                statusList.innerHTML = html
            } else {
                statusList.innerHTML = '<div class="flex items-center justify-center py-4"><span class="text-xs text-red-500">加载失败</span></div>'
            }
        } catch (err) {
            // Ignore runtime.lastError from browser extensions
            if (err.message && err.message.includes('runtime.lastError')) {
                return
            }
            console.error('Failed to load system status:', err)
            let statusList = document.getElementById('system_status_list')
            if (statusList) {
                statusList.innerHTML = '<div class="flex items-center justify-center py-4"><span class="text-xs text-red-500">加载失败</span></div>'
            }
        }
    },

    showChangePassword() {
        let html = `
            <div class="fixed inset-0 z-50 overflow-y-auto" id="change_password_modal">
                <div class="flex min-h-full items-center justify-center p-4">
                    <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onclick="document.getElementById('change_password_modal').remove()"></div>
                    <div class="relative transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg">
                        <div class="bg-white px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
                            <div class="sm:flex sm:items-start">
                                <div class="mt-3 text-center sm:mt-0 sm:text-left w-full">
                                    <h3 class="text-lg font-semibold leading-6 text-gray-900 mb-4">修改密码</h3>
                                    <form id="change_password_form" onsubmit="event.preventDefault(); window.adminAppInstance.handleChangePasswordSubmit(event);">
                                        <div class="space-y-4">
                                            <div>
                                                <label for="current_password" class="block text-sm font-medium text-gray-700 mb-1">当前密码</label>
                                                <input type="password" id="current_password" name="currentPassword" required
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                            </div>
                                            <div>
                                                <label for="new_password" class="block text-sm font-medium text-gray-700 mb-1">新密码</label>
                                                <input type="password" id="new_password" name="newPassword" required minlength="6"
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                                <p class="mt-1 text-xs text-gray-500">密码长度至少为6位</p>
                                            </div>
                                            <div>
                                                <label for="confirm_password" class="block text-sm font-medium text-gray-700 mb-1">确认新密码</label>
                                                <input type="password" id="confirm_password" name="confirmPassword" required minlength="6"
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                            </div>
                                        </div>
                                        <div class="mt-5 sm:mt-6 sm:flex sm:flex-row-reverse gap-3">
                                            <button type="submit" 
                                                    class="inline-flex w-full justify-center rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 sm:ml-3 sm:w-auto">
                                                确认修改
                                            </button>
                                            <button type="button" onclick="document.getElementById('change_password_modal').remove()"
                                                    class="mt-3 inline-flex w-full justify-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 sm:mt-0 sm:w-auto">
                                                取消
                                            </button>
                                        </div>
                                    </form>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `
        document.body.insertAdjacentHTML('beforeend', html)
    },

    handleChangePasswordSubmit(event) {
        let form = document.getElementById('change_password_form')
        let formData = new FormData(form)
        let currentPassword = formData.get('currentPassword')
        let newPassword = formData.get('newPassword')
        let confirmPassword = formData.get('confirmPassword')
        
        if (!currentPassword || !newPassword || !confirmPassword) {
            alert('请填写所有字段')
            return
        }
        
        if (newPassword !== confirmPassword) {
            alert('两次输入的密码不一致')
            return
        }
        
        if (newPassword.length < 6) {
            alert('密码长度至少为6位')
            return
        }
        
        this.changePassword(currentPassword, newPassword)
        document.getElementById('change_password_modal').remove()
    },

    async changePassword(currentPassword, newPassword) {
        try {
            let resp = await fetch('/api/auth/change-password', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    currentPassword: currentPassword,
                    newPassword: newPassword,
                    confirmPassword: newPassword
                }),
                cache: "no-store",
            })
            
            let data = await resp.json()
            if (data.success || data.code === 200) {
                alert('密码修改成功，请重新登录')
                if (data.data && data.data.logout) {
                    window.location.href = '/api/auth/logout?next=/api/auth/login'
                } else {
                    setTimeout(() => {
                        window.location.href = '/api/auth/logout?next=/api/auth/login'
                    }, 1000)
                }
            } else {
                alert('密码修改失败: ' + (data.error || data.msg || '未知错误'))
            }
        } catch (err) {
            alert('密码修改失败: ' + err.message)
        }
    },

    showEditProfile() {
        let user = this.user || {}
        let escapeHtml = (str) => {
            if (!str) return ''
            return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#039;')
        }
        let firstName = escapeHtml(user.firstName || '')
        let lastName = escapeHtml(user.lastName || '')
        let displayName = escapeHtml(user.displayName || (firstName + ' ' + lastName).trim() || '')
        let email = escapeHtml(user.email || '')
        
        let html = `
            <div class="fixed inset-0 z-50 overflow-y-auto" id="edit_profile_modal">
                <div class="flex min-h-full items-center justify-center p-4">
                    <div class="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onclick="document.getElementById('edit_profile_modal').remove()"></div>
                    <div class="relative transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg">
                        <div class="bg-white px-4 pb-4 pt-5 sm:p-6 sm:pb-4">
                            <div class="sm:flex sm:items-start">
                                <div class="mt-3 text-center sm:mt-0 sm:text-left w-full">
                                    <h3 class="text-lg font-semibold leading-6 text-gray-900 mb-4">编辑资料</h3>
                                    <form id="edit_profile_form" onsubmit="event.preventDefault(); window.adminAppInstance.handleEditProfileSubmit(event);">
                                        <div class="space-y-4">
                                            <div>
                                                <label for="first_name" class="block text-sm font-medium text-gray-700 mb-1">名字</label>
                                                <input type="text" id="first_name" name="firstName" value="${firstName}"
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                            </div>
                                            <div>
                                                <label for="last_name" class="block text-sm font-medium text-gray-700 mb-1">姓氏</label>
                                                <input type="text" id="last_name" name="lastName" value="${lastName}"
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                            </div>
                                            <div>
                                                <label for="display_name" class="block text-sm font-medium text-gray-700 mb-1">显示名称</label>
                                                <input type="text" id="display_name" name="displayName" value="${displayName}"
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                            </div>
                                            <div>
                                                <label for="email" class="block text-sm font-medium text-gray-700 mb-1">邮箱</label>
                                                <input type="email" id="email" name="email" value="${email}"
                                                       class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 text-sm">
                                            </div>
                                        </div>
                                        <div class="mt-5 sm:mt-6 sm:flex sm:flex-row-reverse gap-3">
                                            <button type="submit" 
                                                    class="inline-flex w-full justify-center rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 sm:ml-3 sm:w-auto">
                                                保存
                                            </button>
                                            <button type="button" onclick="document.getElementById('edit_profile_modal').remove()"
                                                    class="mt-3 inline-flex w-full justify-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 sm:mt-0 sm:w-auto">
                                                取消
                                            </button>
                                        </div>
                                    </form>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `
        document.body.insertAdjacentHTML('beforeend', html)
    },

    handleEditProfileSubmit(event) {
        let form = document.getElementById('edit_profile_form')
        let formData = new FormData(form)
        let profileData = {
            firstName: formData.get('firstName') || '',
            lastName: formData.get('lastName') || '',
            displayName: formData.get('displayName') || '',
            email: formData.get('email') || ''
        }
        
        this.updateProfile(profileData)
        document.getElementById('edit_profile_modal').remove()
    },

    async updateProfile(profileData) {
        try {
            let resp = await fetch('/api/auth/update', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(profileData),
                cache: "no-store",
            })
            
            let data = await resp.json()
            if (data.success || data.code === 200) {
                alert('资料更新成功')
                if (data.data) {
                    this.user = Object.assign(this.user || {}, data.data)
                    if (data.data.displayName) {
                        this.user.name = data.data.displayName
                    } else if (data.data.firstName || data.data.lastName) {
                        this.user.name = (data.data.firstName || '') + ' ' + (data.data.lastName || '')
                    }
                }
                this.showProfile()
            } else {
                alert('资料更新失败: ' + (data.error || data.msg || '未知错误'))
            }
        } catch (err) {
            alert('资料更新失败: ' + err.message)
        }
    },

    loadAllScripts(objects) {
        objects.forEach(obj => {
            let scripts = obj.scripts || []
            scripts.forEach(s => {
                if (s.onload || this.loadScripts[s.src]) {
                    return
                }
                this.loadScripts[s.src] = true
                let sel = document.createElement('script')
                sel.src = s.src
                sel.defer = true
                document.head.appendChild(sel)
            })
            let styles = obj.styles || []
            styles.forEach(s => {
                if (this.loadStyles[s]) {
                    return
                }
                this.loadStyles[s] = true
                let sel = document.createElement('link')
                sel.rel = 'stylesheet'
                sel.type = 'text/css'
                sel.href = s
                document.head.appendChild(sel)
            })
        })
    },
    onLoad() {
        if (this.$router.path) {
            // switch to current object
            let obj = this.$store.objects.find(obj => obj.path === this.$router.path)
            if (obj) {
                this.switchObject(null, obj)
            }
        } else {
            if (this.site.dashboard) {
                fetch(this.site.dashboard, {
                    cache: "no-store",
                }).then(resp => {
                    this.$store.switching = true
                    resp.text().then(text => {
                        if (text && text.trim()) {
                            let elm = document.getElementById('query_content')
                            this.injectHtml(elm, text, null)
                            this.$store.switching = false
                        } else {
                            // Dashboard returned empty, show default image
                            this.showDefaultImage()
                        }
                    })
                }).catch(() => {
                    // If dashboard fetch fails, show default image
                    this.showDefaultImage()
                })
            } else {
                // No dashboard configured, show default image
                this.showDefaultImage()
            }
        }
    },
    showDefaultImage() {
        this.$store.switching = false
        // Ensure editobj.mode is undefined so query_content is visible
        if (this.$store.editobj) {
            this.$store.editobj.mode = undefined
        }
        
        // Use requestAnimationFrame to ensure DOM is ready
        requestAnimationFrame(() => {
            let elm = document.getElementById('query_content')
            if (!elm) {
                // Retry after a short delay
                setTimeout(() => this.showDefaultImage(), 200)
                return
            }
            
            // Get monitor URL from site config
            let monitorUrl = (this.site?.Site?.monitor_url) || (this.site?.monitor_url) || ''
            
            // Get user info
            let userName = this.user?.name || this.user?.firstName || this.user?.email || '管理员'
            let currentTime = new Date().toLocaleString('zh-CN', { 
                year: 'numeric', 
                month: 'long', 
                day: 'numeric', 
                hour: '2-digit', 
                minute: '2-digit',
                weekday: 'long'
            })
            
            // Load dashboard metrics from backend
            this.loadDashboardMetrics(elm, monitorUrl, userName, currentTime)
        })
    },
    
    async loadDashboardMetrics(elm, monitorUrl, userName, currentTime) {
        // Default mock data (fallback)
        let mockData = {
            pv: { today: 0, yesterday: 0, change: 0.0 },
            uv: { today: 0, yesterday: 0, change: 0.0 },
            apiCalls: { today: 0, yesterday: 0, change: 0.0 },
            activeUsers: { today: 0, yesterday: 0, change: 0.0 },
            responseTime: { avg: 125, p95: 234, p99: 456 },
            errorRate: { today: 0.12, yesterday: 0.15, change: -20.0 },
            throughput: { today: 1250, yesterday: 1180, change: 5.9 }
        }
        
        try {
            let resp = await fetch('/api/system/dashboard/metrics', {
                method: 'GET',
                cache: "no-store",
            })
            if (resp.ok) {
                let data = await resp.json()
                if (data.success && data.data) {
                    // Use real data from backend
                    mockData.pv = data.data.pv || mockData.pv
                    mockData.uv = data.data.uv || mockData.uv
                    mockData.apiCalls = data.data.apiCalls || mockData.apiCalls
                    mockData.activeUsers = data.data.activeUsers || mockData.activeUsers
                }
            }
        } catch (err) {
            console.error('Failed to load dashboard metrics:', err)
        }
        
        // Render dashboard with data
        this.renderDashboard(elm, monitorUrl, userName, currentTime, mockData)
    },
    
    renderDashboard(elm, monitorUrl, userName, currentTime, mockData) {
        let html = '<div class="space-y-6">'
            
            // Welcome Section
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-6">'
            html += '<div class="flex items-center justify-between flex-wrap gap-4">'
            html += '<div>'
            html += '<h1 class="text-2xl font-semibold text-gray-900 mb-1">欢迎回来，' + userName + '</h1>'
            html += '<p class="text-sm text-gray-500">' + currentTime + '</p>'
            html += '</div>'
            html += '<a href="#" onclick="if(window.adminAppInstance){window.adminAppInstance.showProfile(event);} return false;" class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 bg-gray-50 border border-gray-300 rounded-md hover:bg-gray-100 transition-colors">'
            html += '<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path></svg>'
            html += '<span>个人中心</span>'
            html += '</a>'
            html += '</div>'
            html += '</div>'
            
            // Quick Actions Section
            html += '<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">'
            // Always show monitor link, construct URL if not provided
            let finalMonitorUrl = monitorUrl || '/api/metrics/ui'
            html += '<a href="' + finalMonitorUrl + '" target="_blank" class="group bg-white border border-gray-200 rounded-lg shadow-sm p-6 hover:border-gray-300 hover:shadow-md transition-all">'
            html += '<div class="flex items-center justify-between mb-4">'
            html += '<div class="p-2 bg-gray-100 rounded-md">'
            html += '<svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path></svg>'
            html += '</div>'
            html += '<svg class="w-4 h-4 text-gray-400 group-hover:text-gray-600 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path></svg>'
            html += '</div>'
            html += '<h3 class="text-base font-semibold text-gray-900 mb-1">性能监控</h3>'
            html += '<p class="text-sm text-gray-500">查看系统性能指标和监控数据</p>'
            html += '</a>'
            html += '</div>'
            
            // Key Metrics Section
            html += '<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">'
            
            // PV Card
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5 hover:border-gray-300 transition-colors">'
            html += '<div class="flex items-center justify-between mb-3">'
            html += '<div class="p-1.5 bg-gray-100 rounded">'
            html += '<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path></svg>'
            html += '</div>'
            html += '<span class="text-xs font-medium px-2 py-0.5 rounded ' + (mockData.pv.change > 0 ? 'bg-gray-100 text-gray-700' : 'bg-gray-100 text-gray-700') + '">' + (mockData.pv.change > 0 ? '+' : '') + mockData.pv.change.toFixed(1) + '%</span>'
            html += '</div>'
            html += '<div class="text-xl font-semibold text-gray-900 mb-1">' + mockData.pv.today.toLocaleString() + '</div>'
            html += '<div class="text-xs text-gray-500">页面浏览量 (PV)</div>'
            html += '<div class="text-xs text-gray-400 mt-1">昨日: ' + mockData.pv.yesterday.toLocaleString() + '</div>'
            html += '</div>'
            
            // UV Card
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5 hover:border-gray-300 transition-colors">'
            html += '<div class="flex items-center justify-between mb-3">'
            html += '<div class="p-1.5 bg-gray-100 rounded">'
            html += '<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"></path></svg>'
            html += '</div>'
            html += '<span class="text-xs font-medium px-2 py-0.5 rounded ' + (mockData.uv.change > 0 ? 'bg-gray-100 text-gray-700' : 'bg-gray-100 text-gray-700') + '">' + (mockData.uv.change > 0 ? '+' : '') + mockData.uv.change.toFixed(1) + '%</span>'
            html += '</div>'
            html += '<div class="text-xl font-semibold text-gray-900 mb-1">' + mockData.uv.today.toLocaleString() + '</div>'
            html += '<div class="text-xs text-gray-500">独立访客 (UV)</div>'
            html += '<div class="text-xs text-gray-400 mt-1">昨日: ' + mockData.uv.yesterday.toLocaleString() + '</div>'
            html += '</div>'
            
            // API Calls Card
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5 hover:border-gray-300 transition-colors">'
            html += '<div class="flex items-center justify-between mb-3">'
            html += '<div class="p-1.5 bg-gray-100 rounded">'
            html += '<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>'
            html += '</div>'
            html += '<span class="text-xs font-medium px-2 py-0.5 rounded ' + (mockData.apiCalls.change > 0 ? 'bg-gray-100 text-gray-700' : 'bg-gray-100 text-gray-700') + '">' + (mockData.apiCalls.change > 0 ? '+' : '') + mockData.apiCalls.change.toFixed(1) + '%</span>'
            html += '</div>'
            html += '<div class="text-xl font-semibold text-gray-900 mb-1">' + mockData.apiCalls.today.toLocaleString() + '</div>'
            html += '<div class="text-xs text-gray-500">API 调用次数</div>'
            html += '<div class="text-xs text-gray-400 mt-1">昨日: ' + mockData.apiCalls.yesterday.toLocaleString() + '</div>'
            html += '</div>'
            
            // Active Users Card
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5 hover:border-gray-300 transition-colors">'
            html += '<div class="flex items-center justify-between mb-3">'
            html += '<div class="p-1.5 bg-gray-100 rounded">'
            html += '<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path></svg>'
            html += '</div>'
            html += '</div>'
            html += '<div class="text-xl font-semibold text-gray-900 mb-1">' + mockData.activeUsers.today.toLocaleString() + '</div>'
            html += '<div class="text-xs text-gray-500">活跃用户</div>'
            html += '<div class="text-xs text-gray-400 mt-1">最近24小时</div>'
            html += '</div>'
            
            html += '</div>'
            
            // System Status Section
            html += '<div class="bg-white border border-gray-200 rounded-lg shadow-sm p-5">'
            html += '<h3 class="text-sm font-semibold text-gray-900 mb-3">系统状态</h3>'
            html += '<div class="space-y-2" id="system_status_list">'
            html += '<div class="flex items-center justify-center py-4">'
            html += '<div class="animate-spin inline-block w-4 h-4 border-2 border-gray-300 border-t-gray-600 rounded-full"></div>'
            html += '<span class="ml-2 text-xs text-gray-500">检查中...</span>'
            html += '</div>'
            html += '</div>'
            html += '</div>'
            
            html += '</div>'
            
            elm.innerHTML = html
            // Force display in case x-show is hiding it
            elm.style.display = 'block'
            Alpine.initTree(elm)
            
            // Make instance globally accessible
            window.adminAppInstance = this
            
        // Load system status after DOM is ready
        setTimeout(() => {
            this.loadSystemStatus()
        }, 200)
    },
    loadSidebar() {
        fetch('sidebar.html', {
            cache: "no-store",
        }).then(resp => {
            resp.text().then(text => {
                if (text) {
                    this.injectHtml(this.$refs.sidebar, text, null)
                }
            })
        })
    },

    buildNavMenu() {
        let menus = []
        this.$store.objects.forEach(obj => {
            if (obj.invisible) { // skip invisible object
                return
            }
            let menu = menus.find(m => m.name === obj.group)
            if (!menu) {
                menu = { name: obj.group, items: [] }
                menus.push(menu)
            }
            menu.items.push(obj)
        });
        this.navmenus = menus
    },

    switchObject(event, obj) {
        if (event) {
            event.preventDefault()
        }

        if (this.$store.current) {
            // reset selected filters
            if (this.$store.current.filterables) {
                this.$store.current.filterables.forEach(f => {
                    f.selected = undefined
                })
            }
            if (this.$store.current === obj) return
        }
        this.closeEdit()

        this.$store.queryresult.reset()
        this.$store.switching = true
        this.$store.current = obj
        this.$router.push(obj.path)

        fetch(`/api/admin/${obj.listpage}`, {
            cache: "no-store",
        }).then(resp => {
            resp.text().then(text => {
                const elm = document.getElementById('query_content')
                this.$store.queryresult.setOrders(obj.orders)
                if (!this.injectHtml(elm, text, obj)) {
                    this.$store.queryresult.refresh()
                }
                this.$store.switching = false
            })
        })
    },
    injectHtml(elm, html, obj) {
        let hasOnload = false
        if (obj) {
            let scripts = obj.scripts || []
            scripts.filter(s => s.onload).forEach(s => {
                hasOnload = true
                let sel = document.createElement('script')
                sel.src = s.src
                sel.defer = true
                document.head.appendChild(sel)
            })
        }
        elm.innerHTML = html
        return hasOnload
    },
    prepareEditobj(event, isCreate = false, row = undefined) {
        if (event) {
            event.preventDefault()
        }

        let names = {}
        let fields = this.$store.current.editables.map(editField => {
            let f = { ...editField }
            if (isCreate) {
                f.value = editField.defaultValue()
            } else {
                f.value = row.rawData[editField.name]
            }
            if (f.value && f.foreign) {
                f.value = f.value.value
            }
            names[editField.name] = f
            return f
        })

        let editobj = new EditObject(
            {
                mode: isCreate ? 'create' : 'edit',
                title: this.$store.current.editTitle || `${isCreate ? 'Add' : 'Edit'} ${this.$store.current.name}`,
                fields: fields,
                names,
                primaryValue: row ? row.primaryValue : undefined,
                row
            })

        let current = this.$store.current
        if (current.prepareEdit) {
            current.prepareEdit(editobj, isCreate, row)
        }

        fetch(`/api/admin/${current.editpage}`, {
            cache: "no-store",
        }).then(resp => {
            resp.text().then(text => {
                let elm = document.getElementById('edit_form')
                if (elm) {
                    this.$store.editobj = editobj
                    elm.innerHTML = text
                }
            })
        }).catch(err => {
            Alpine.store('toasts').error(`Load edit page fail: ${err.toString()}`)
        })
    },
    addObject(event) {
        this.prepareEditobj(event, true)
    },
    editObject(event, row) {
        this.prepareEditobj(event, false, row)
    },
    closeEdit(event, cancel = false) {
        if (event) {
            event.preventDefault()
        }

        let elm = document.getElementById('edit_form')
        if (elm) {
            elm.innerHTML = ''
        }
        if (this.$store.editobj) {
            this.$store.editobj.closeEdit(event, cancel)
        }
    },
})

// Make adminapp globally accessible for Alpine.js
window.adminapp = adminapp