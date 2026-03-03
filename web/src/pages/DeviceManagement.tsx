import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import { getAssistantList, AssistantListItem } from '@/api/assistant';
import {
    bindDevice,
    getUserDevices,
    unbindDevice,
    updateDevice,
    manualAddDevice,
    type Device
} from '@/api/device';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import {
    Smartphone,
    Plus,
    Trash2,
    Edit2,
    Key,
    CheckCircle,
    XCircle,
    Cpu,
    Calendar,
    Bot,
    Settings,
    Activity,
    Clock,
    AlertTriangle,
    Thermometer,
    MemoryStick,
    Headphones,
    Mic,
    Eye
} from 'lucide-react';
import Button from '@/components/UI/Button';
import Input from '@/components/UI/Input';
import Modal from '@/components/UI/Modal';
import Card from '@/components/UI/Card';
import ConfirmDialog from '@/components/UI/ConfirmDialog';
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select';
import EmptyState from '@/components/UI/EmptyState';
import Badge from '@/components/UI/Badge';

const DeviceManagement: React.FC = () => {
    const { t } = useI18nStore();
    const navigate = useNavigate();
    const [assistants, setAssistants] = useState<AssistantListItem[]>([]);
    const [selectedAssistantId, setSelectedAssistantId] = useState<string | null>(null);
    const [devices, setDevices] = useState<Device[]>([]);
    const [isLoading, setIsLoading] = useState(false);

    // 绑定设备相关状态
    const [showBindModal, setShowBindModal] = useState(false);
    const [activationCode, setActivationCode] = useState('');
    const [isBinding, setIsBinding] = useState(false);

    // 手动添加设备相关状态
    const [showAddModal, setShowAddModal] = useState(false);
    const [addForm, setAddForm] = useState({
        macAddress: '',
        board: '',
        appVersion: '1.0.0',
    });
    const [isAdding, setIsAdding] = useState(false);

    // 编辑设备相关状态
    const [editingDevice, setEditingDevice] = useState<Device | null>(null);
    const [editForm, setEditForm] = useState({
        alias: '',
        autoUpdate: 1,
    });

    // 解绑确认对话框
    const [showUnbindDialog, setShowUnbindDialog] = useState(false);
    const [deviceToUnbind, setDeviceToUnbind] = useState<Device | null>(null);

    // 获取助手列表
    useEffect(() => {
        const fetchAssistants = async () => {
            try {
                const res = await getAssistantList();
                if (res.code === 200) {
                    setAssistants(res.data);
                    if (res.data.length > 0 && !selectedAssistantId) {
                        setSelectedAssistantId(String(res.data[0].id));
                    }
                } else {
                    showAlert(t('device.messages.fetchAssistantsFailed'), 'error');
                }
            } catch (err: any) {
                showAlert(err?.msg || err?.message || t('device.messages.fetchAssistantsFailed'), 'error');
            }
        };
        fetchAssistants();
    }, []);

    // 获取设备列表
    useEffect(() => {
        if (selectedAssistantId) {
            fetchDevices(selectedAssistantId);
        }
    }, [selectedAssistantId]);

    const fetchDevices = async (assistantId: string) => {
        try {
            setIsLoading(true);
            const res = await getUserDevices(assistantId);
            if (res.code === 200) {
                setDevices(res.data);
            } else {
                showAlert(res.msg || t('device.messages.fetchDevicesFailed'), 'error');
            }
        } catch (err: any) {
            showAlert(err?.msg || err?.message || t('device.messages.fetchDevicesFailed'), 'error');
        } finally {
            setIsLoading(false);
        }
    };

    // 绑定设备
    const handleBindDevice = async () => {
        if (!selectedAssistantId) {
            showAlert(t('device.messages.selectAssistant'), 'error');
            return;
        }
        if (!activationCode.trim()) {
            showAlert(t('device.messages.enterActivationCode'), 'error');
            return;
        }

        setIsBinding(true);
        try {
            const res = await bindDevice(selectedAssistantId, activationCode.trim());
            if (res.code === 200) {
                showAlert(t('device.messages.bindSuccess'), 'success');
                setShowBindModal(false);
                setActivationCode('');
                fetchDevices(selectedAssistantId);
            } else {
                showAlert(res.msg || t('device.messages.bindFailed'), 'error');
            }
        } catch (err: any) {
            showAlert(err?.msg || err?.message || t('device.messages.bindFailed'), 'error');
        } finally {
            setIsBinding(false);
        }
    };

    // 解绑设备
    const handleUnbindDevice = async () => {
        if (!deviceToUnbind) return;

        try {
            const res = await unbindDevice({ deviceId: deviceToUnbind.id });
            if (res.code === 200) {
                showAlert(t('device.messages.unbindSuccess'), 'success');
                if (selectedAssistantId) {
                    fetchDevices(selectedAssistantId);
                }
            } else {
                showAlert(res.msg || t('device.messages.unbindFailed'), 'error');
            }
        } catch (err: any) {
            showAlert(err?.msg || err?.message || t('device.messages.unbindFailed'), 'error');
        } finally {
            setShowUnbindDialog(false);
            setDeviceToUnbind(null);
        }
    };

    // 手动添加设备
    const handleManualAdd = async () => {
        if (!selectedAssistantId) {
            showAlert(t('device.messages.selectAssistant'), 'error');
            return;
        }
        if (!addForm.macAddress.trim()) {
            showAlert(t('device.messages.enterMacAddress'), 'error');
            return;
        }
        if (!addForm.board.trim()) {
            showAlert(t('device.messages.enterBoardType'), 'error');
            return;
        }

        // 验证MAC地址格式
        const macPattern = /^([0-9A-Za-z]{2}[:-]){5}([0-9A-Za-z]{2})$/;
        if (!macPattern.test(addForm.macAddress)) {
            showAlert(t('device.messages.invalidMacAddress'), 'error');
            return;
        }

        setIsAdding(true);
        try {
            const res = await manualAddDevice({
                agentId: selectedAssistantId,
                macAddress: addForm.macAddress.trim(),
                board: addForm.board.trim(),
                appVersion: addForm.appVersion || '1.0.0',
            });
            if (res.code === 200) {
                showAlert(t('device.messages.manualAddSuccess'), 'success');
                setShowAddModal(false);
                setAddForm({ macAddress: '', board: '', appVersion: '1.0.0' });
                fetchDevices(selectedAssistantId);
            } else {
                showAlert(res.msg || t('device.messages.manualAddFailed'), 'error');
            }
        } catch (err: any) {
            showAlert(err?.msg || err?.message || t('device.messages.manualAddFailed'), 'error');
        } finally {
            setIsAdding(false);
        }
    };

    // 更新设备信息
    const handleUpdateDevice = async () => {
        if (!editingDevice) return;

        try {
            const res = await updateDevice(editingDevice.id, editForm);
            if (res.code === 200) {
                showAlert(t('device.messages.updateSuccess'), 'success');
                setEditingDevice(null);
                if (selectedAssistantId) {
                    fetchDevices(selectedAssistantId);
                }
            } else {
                showAlert(res.msg || t('device.messages.updateFailed'), 'error');
            }
        } catch (err: any) {
            showAlert(err?.msg || err?.message || t('device.messages.updateFailed'), 'error');
        }
    };

    // 打开编辑对话框
    const openEditModal = (device: Device) => {
        setEditingDevice(device);
        setEditForm({
            alias: device.alias || '',
            autoUpdate: device.autoUpdate,
        });
    };

    // 打开解绑确认对话框
    const openUnbindDialog = (device: Device) => {
        setDeviceToUnbind(device);
        setShowUnbindDialog(true);
    };

    const fmtDate = (iso?: string) => (iso ? new Date(iso).toLocaleString() : 'N/A');

    const formatUptime = (seconds: number) => {
        if (!seconds) return 'N/A';
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        if (hours > 0) {
            return `${hours}h ${minutes}m`;
        }
        return `${minutes}m`;
    };

    const getStatusColor = (isOnline: boolean) => {
        return isOnline ? 'text-green-500' : 'text-red-500';
    };

    const getPerformanceColor = (value: number, type: 'cpu' | 'memory' | 'temperature') => {
        if (type === 'temperature') {
            if (value > 80) return 'text-red-500';
            if (value > 60) return 'text-yellow-500';
            return 'text-green-500';
        }
        if (value > 80) return 'text-red-500';
        if (value > 60) return 'text-yellow-500';
        return 'text-green-500';
    };

    return (
        <div className="min-h-screen dark:bg-neutral-900 flex flex-col">
            <div className="max-w-6xl w-full mx-auto pt-10 pb-4 px-4 flex flex-col">
                <div className="flex items-center justify-between mb-7">
                    <div className="relative pl-4">
                        <motion.div
                            layoutId="pageTitleIndicator"
                            className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary rounded-r-full"
                            transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
                        />
                        <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                            {t('device.title')}
                        </h1>
                        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                            {t('device.subtitle')}
                        </p>
                    </div>
                    <div className="flex gap-3">
                        <Button
                            onClick={() => setShowBindModal(true)}
                            variant="primary"
                            size="md"
                            leftIcon={<Key className="w-4 h-4" />}
                        >
                            {t('device.bindDevice')}
                        </Button>
                        <Button
                            onClick={() => setShowAddModal(true)}
                            variant="secondary"
                            size="md"
                            leftIcon={<Plus className="w-4 h-4" />}
                        >
                            {t('device.manualAddDevice')}
                        </Button>
                    </div>
                </div>

                {/* 助手选择 - 使用按钮组 */}
                {assistants.length > 0 ? (
                    <div className="w-full">
                        <div className="flex flex-wrap gap-2 mb-6">
                            {assistants.map(assistant => (
                                <Button
                                    key={assistant.id}
                                    variant={selectedAssistantId === String(assistant.id) ? 'primary' : 'outline'}
                                    size="md"
                                    onClick={() => setSelectedAssistantId(String(assistant.id))}
                                    leftIcon={<Bot className="w-4 h-4" />}
                                    className="flex-shrink-0"
                                >
                                    {assistant.name}
                                </Button>
                            ))}
                        </div>

                        {assistants.map(assistant => (
                            selectedAssistantId === String(assistant.id) && (
                                <div key={assistant.id} className="mt-6">
                                    {isLoading ? (
                                        <EmptyState
                                            icon={Smartphone}
                                            title={t('device.loading')}
                                            iconClassName="text-gray-400 animate-pulse"
                                        />
                                    ) : devices.length === 0 ? (
                                        <EmptyState
                                            icon={Smartphone}
                                            title={t('device.empty')}
                                            description={t('device.emptyDesc')}
                                            action={{
                                                label: t('device.bindDevice'),
                                                onClick: () => setShowBindModal(true)
                                            }}
                                        />
                                    ) : (
                                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-5">
                                            {devices.map(device => (
                                                <Card
                                                    key={device.id}
                                                    variant="outlined"
                                                    hover
                                                    padding="md"
                                                    className="flex flex-col"
                                                >
                                                    <div className="flex items-start justify-between mb-3">
                                                        <div className="flex items-center gap-2 flex-1 min-w-0">
                                                            <div className="relative">
                                                                <Smartphone className="w-5 h-5 text-purple-500 flex-shrink-0" />
                                                                <div className={`absolute -top-1 -right-1 w-3 h-3 rounded-full ${device.isOnline ? 'bg-green-500' : 'bg-red-500'}`} />
                                                            </div>
                                                            <div className="min-w-0 flex-1">
                                                                <h3 className="font-semibold text-lg truncate text-gray-900 dark:text-gray-100">
                                                                    {device.deviceName || device.macAddress}
                                                                </h3>
                                                                {device.deviceName && (
                                                                    <p className="text-xs text-gray-500 dark:text-gray-400 truncate font-mono">
                                                                        {device.macAddress}
                                                                    </p>
                                                                )}
                                                            </div>
                                                        </div>
                                                        <div className="flex gap-1 flex-shrink-0">
                                                            <Button
                                                                variant="ghost"
                                                                size="icon"
                                                                onClick={() => navigate(`/devices/${device.id}`)}
                                                                title="查看详情"
                                                            >
                                                                <Eye className="w-4 h-4" />
                                                            </Button>
                                                            <Button
                                                                variant="ghost"
                                                                size="icon"
                                                                onClick={() => openEditModal(device)}
                                                                title={t('device.edit')}
                                                            >
                                                                <Edit2 className="w-4 h-4" />
                                                            </Button>
                                                            <Button
                                                                variant="ghost"
                                                                size="icon"
                                                                onClick={() => openUnbindDialog(device)}
                                                                title={t('device.unbind')}
                                                            >
                                                                <Trash2 className="w-4 h-4 text-red-500" />
                                                            </Button>
                                                        </div>
                                                    </div>

                                                    {/* 设备状态 */}
                                                    <div className="flex items-center gap-4 mb-3 text-sm">
                                                        <div className="flex items-center gap-1">
                                                            <Activity className={`w-4 h-4 ${getStatusColor(device.isOnline)}`} />
                                                            <span className={getStatusColor(device.isOnline)}>
                              {device.isOnline ? t('device.online') : t('device.offline')}
                            </span>
                                                        </div>
                                                        {device.uptime > 0 && (
                                                            <div className="flex items-center gap-1 text-gray-600 dark:text-gray-400">
                                                                <Clock className="w-4 h-4" />
                                                                <span className="text-xs">{formatUptime(device.uptime)}</span>
                                                            </div>
                                                        )}
                                                        {device.errorCount > 0 && (
                                                            <div className="flex items-center gap-1 text-red-500">
                                                                <AlertTriangle className="w-4 h-4" />
                                                                <span className="text-xs">{device.errorCount}</span>
                                                            </div>
                                                        )}
                                                    </div>

                                                    {/* 性能指标 */}
                                                    {(device.cpuUsage > 0 || device.memoryUsage > 0 || device.temperature > 0) && (
                                                        <div className="grid grid-cols-3 gap-2 mb-3 text-xs">
                                                            {device.cpuUsage > 0 && (
                                                                <div className="flex items-center gap-1">
                                                                    <Cpu className={`w-3 h-3 ${getPerformanceColor(device.cpuUsage, 'cpu')}`} />
                                                                    <span className={getPerformanceColor(device.cpuUsage, 'cpu')}>
                                  {device.cpuUsage.toFixed(1)}%
                                </span>
                                                                </div>
                                                            )}
                                                            {device.memoryUsage > 0 && (
                                                                <div className="flex items-center gap-1">
                                                                    <MemoryStick className={`w-3 h-3 ${getPerformanceColor(device.memoryUsage, 'memory')}`} />
                                                                    <span className={getPerformanceColor(device.memoryUsage, 'memory')}>
                                  {device.memoryUsage.toFixed(1)}%
                                </span>
                                                                </div>
                                                            )}
                                                            {device.temperature > 0 && (
                                                                <div className="flex items-center gap-1">
                                                                    <Thermometer className={`w-3 h-3 ${getPerformanceColor(device.temperature, 'temperature')}`} />
                                                                    <span className={getPerformanceColor(device.temperature, 'temperature')}>
                                  {device.temperature.toFixed(1)}°C
                                </span>
                                                                </div>
                                                            )}
                                                        </div>
                                                    )}

                                                    <div className="space-y-2 text-sm flex-1">
                                                        {device.board && (
                                                            <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                                                                <Cpu className="w-4 h-4 flex-shrink-0" />
                                                                <span className="truncate">{device.board}</span>
                                                            </div>
                                                        )}
                                                        {device.appVersion && (
                                                            <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                                                                <Settings className="w-4 h-4 flex-shrink-0" />
                                                                <span className="text-xs">
                                {t('device.appVersion')}: {device.appVersion}
                              </span>
                                                            </div>
                                                        )}

                                                        {/* 音频设备状态 */}
                                                        {device.audioStatus && (
                                                            <div className="flex items-center gap-2 text-xs">
                                                                <div className="flex items-center gap-1 text-green-500">
                                                                    <Mic className="w-3 h-3" />
                                                                    <span>Mic</span>
                                                                </div>
                                                                <div className="flex items-center gap-1 text-blue-500">
                                                                    <Headphones className="w-3 h-3" />
                                                                    <span>Speaker</span>
                                                                </div>
                                                            </div>
                                                        )}

                                                        <div className="flex items-center gap-2">
                                                            <Badge
                                                                variant={device.autoUpdate === 1 ? 'success' : 'muted'}
                                                                size="sm"
                                                                icon={device.autoUpdate === 1 ? <CheckCircle className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                                                            >
                                                                {device.autoUpdate === 1 ? t('device.autoUpdate') + ': ' + t('device.enabled') : t('device.autoUpdate') + ': ' + t('device.disabled')}
                                                            </Badge>
                                                        </div>

                                                        {device.lastSeen && (
                                                            <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                                                                <Calendar className="w-4 h-4 flex-shrink-0" />
                                                                <span>{t('device.lastSeen')}: {fmtDate(device.lastSeen)}</span>
                                                            </div>
                                                        )}
                                                    </div>
                                                </Card>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            )
                        ))}
                    </div>
                ) : (
                    <EmptyState
                        icon={Bot}
                        title={t('device.emptyNoAssistant')}
                        description={t('assistants.empty')}
                    />
                )}

                {/* 绑定设备模态框 */}
                <Modal
                    isOpen={showBindModal}
                    onClose={() => {
                        setShowBindModal(false);
                        setActivationCode('');
                    }}
                    title={t('device.bindModal.title')}
                >
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                {t('device.bindModal.activationCode')}
                            </label>
                            <Input
                                type="text"
                                value={activationCode}
                                onChange={(e) => setActivationCode(e.target.value)}
                                placeholder={t('device.bindModal.activationCodePlaceholder')}
                                maxLength={6}
                                className="w-full"
                            />
                            <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                                {t('device.bindModal.activationCodeDesc')}
                            </p>
                        </div>
                        <div className="flex justify-end gap-2">
                            <Button
                                variant="outline"
                                onClick={() => {
                                    setShowBindModal(false);
                                    setActivationCode('');
                                }}
                            >
                                {t('device.cancel')}
                            </Button>
                            <Button
                                variant="primary"
                                onClick={handleBindDevice}
                                disabled={isBinding || !activationCode.trim()}
                            >
                                {isBinding ? t('device.bindModal.binding') : t('device.bindDevice')}
                            </Button>
                        </div>
                    </div>
                </Modal>

                {/* 手动添加设备模态框 */}
                <Modal
                    isOpen={showAddModal}
                    onClose={() => {
                        setShowAddModal(false);
                        setAddForm({ macAddress: '', board: '', appVersion: '1.0.0' });
                    }}
                    title={t('device.addModal.title')}
                >
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                {t('device.addModal.macAddress')} <span className="text-red-500">*</span>
                            </label>
                            <Input
                                type="text"
                                value={addForm.macAddress}
                                onChange={(e) => setAddForm({ ...addForm, macAddress: e.target.value })}
                                placeholder={t('device.addModal.macAddressPlaceholder')}
                                className="w-full font-mono"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                {t('device.addModal.board')} <span className="text-red-500">*</span>
                            </label>
                            <Input
                                type="text"
                                value={addForm.board}
                                onChange={(e) => setAddForm({ ...addForm, board: e.target.value })}
                                placeholder={t('device.addModal.boardPlaceholder')}
                                className="w-full"
                            />
                        </div>
                        <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                {t('device.addModal.appVersion')}
                            </label>
                            <Input
                                type="text"
                                value={addForm.appVersion}
                                onChange={(e) => setAddForm({ ...addForm, appVersion: e.target.value })}
                                placeholder={t('device.addModal.appVersionPlaceholder')}
                                className="w-full"
                            />
                        </div>
                        <div className="flex justify-end gap-2">
                            <Button
                                variant="outline"
                                onClick={() => {
                                    setShowAddModal(false);
                                    setAddForm({ macAddress: '', board: '', appVersion: '1.0.0' });
                                }}
                            >
                                {t('device.cancel')}
                            </Button>
                            <Button
                                variant="primary"
                                onClick={handleManualAdd}
                                disabled={isAdding}
                            >
                                {isAdding ? t('device.addModal.adding') : t('device.manualAddDevice')}
                            </Button>
                        </div>
                    </div>
                </Modal>

                {/* 编辑设备模态框 */}
                <Modal
                    isOpen={!!editingDevice}
                    onClose={() => setEditingDevice(null)}
                    title={t('device.editModal.title')}
                >
                    {editingDevice && (
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    {t('device.alias')}
                                </label>
                                <Input
                                    type="text"
                                    value={editForm.alias}
                                    onChange={(e) => setEditForm({ ...editForm, alias: e.target.value })}
                                    placeholder={t('device.editModal.aliasPlaceholder')}
                                    className="w-full"
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                                    {t('device.editModal.autoUpdateLabel')}
                                </label>
                                <Select
                                    value={String(editForm.autoUpdate)}
                                    onValueChange={(value) => setEditForm({ ...editForm, autoUpdate: parseInt(value) })}
                                >
                                    <SelectTrigger className="w-full">
                                        <SelectValue placeholder={t('device.editModal.autoUpdateLabel')} />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="1">{t('device.enabled')}</SelectItem>
                                        <SelectItem value="0">{t('device.disabled')}</SelectItem>
                                    </SelectContent>
                                </Select>
                            </div>
                            <div className="flex justify-end gap-2">
                                <Button
                                    variant="outline"
                                    onClick={() => setEditingDevice(null)}
                                >
                                    {t('device.cancel')}
                                </Button>
                                <Button
                                    variant="primary"
                                    onClick={handleUpdateDevice}
                                >
                                    {t('device.save')}
                                </Button>
                            </div>
                        </div>
                    )}
                </Modal>

                {/* 确认解绑对话框 */}
                <ConfirmDialog
                    isOpen={showUnbindDialog}
                    onClose={() => {
                        setShowUnbindDialog(false);
                        setDeviceToUnbind(null);
                    }}
                    onConfirm={handleUnbindDevice}
                    title={t('device.messages.confirmUnbindTitle')}
                    message={t('device.messages.confirmUnbindDescription', {
                        deviceName : deviceToUnbind?.alias || deviceToUnbind?.macAddress || '未知设备'
                    })}
                    confirmText={t('device.messages.confirmUnbind')}
                    cancelText={t('device.cancel')}
                    type="danger"
                />
            </div>
        </div>
    );
};

export default DeviceManagement;