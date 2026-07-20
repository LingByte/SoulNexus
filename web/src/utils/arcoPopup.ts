import type { ConfigProviderProps } from '@arco-design/web-react'

/** Above modal (1001), sidebar (90), call-flow drawer (1050). */
export const ARCO_POPUP_Z_INDEX = 11000

export const arcoPopupContainer = (): HTMLElement => document.body

const triggerPopup = {
  autoAlignPopupWidth: true,
  position: 'bl' as const,
  updateOnScroll: true,
  popupStyle: { zIndex: ARCO_POPUP_Z_INDEX },
}

/** Merged into Arco ConfigProvider — fixes Select / Dropdown popups app-wide. */
export const arcoGlobalComponentConfig: NonNullable<ConfigProviderProps['componentConfig']> = {
  Select: {
    getPopupContainer: arcoPopupContainer,
    triggerProps: triggerPopup,
    dropdownMenuStyle: { zIndex: ARCO_POPUP_Z_INDEX },
  },
  Cascader: {
    getPopupContainer: arcoPopupContainer,
    triggerProps: triggerPopup,
  },
  TreeSelect: {
    getPopupContainer: arcoPopupContainer,
    triggerProps: triggerPopup,
  },
  AutoComplete: {
    getPopupContainer: arcoPopupContainer,
    triggerProps: triggerPopup,
  },
  DatePicker: {
    getPopupContainer: arcoPopupContainer,
  },
  TimePicker: {
    getPopupContainer: arcoPopupContainer,
  },
  Dropdown: {
    getPopupContainer: arcoPopupContainer,
  },
  Tooltip: {
    getPopupContainer: arcoPopupContainer,
  },
  Popover: {
    getPopupContainer: arcoPopupContainer,
  },
}
