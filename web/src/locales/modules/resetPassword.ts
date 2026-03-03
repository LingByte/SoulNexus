import { Language } from './common'

export const resetPassword: Record<Language, Record<string, string>> = {
  zh: {
    // Reset Password 页面
    'resetPassword.title': '重置密码',
    'resetPassword.subtitle': '请输入您的新密码',
    'resetPassword.newPassword': '新密码',
    'resetPassword.newPasswordPlaceholder': '请输入新密码（至少6位）',
    'resetPassword.confirmPassword': '确认密码',
    'resetPassword.confirmPasswordPlaceholder': '请再次输入新密码',
    'resetPassword.submit': '重置密码',
    'resetPassword.submitting': '重置中...',
    'resetPassword.backToHome': '返回首页',
    'resetPassword.loginNow': '立即登录',
    
    // Invalid Token
    'resetPassword.invalid.title': '链接无效',
    'resetPassword.invalid.description': '重置密码链接无效或已过期。请重新申请密码重置。',
    
    // Success
    'resetPassword.success.title': '重置成功！',
    'resetPassword.success.description': '您的密码已成功重置。正在跳转到登录页面...',
    
    // Messages
    'resetPassword.messages.invalidLink': '重置链接无效或已过期',
    'resetPassword.messages.linkInvalid': '链接无效',
    'resetPassword.messages.tokenInvalid': '重置链接无效',
    'resetPassword.messages.verifyFailed': '验证失败',
    'resetPassword.messages.passwordMismatch': '密码不匹配',
    'resetPassword.messages.passwordTooShort': '密码至少需要6位',
    'resetPassword.messages.resetSuccess': '密码重置成功！',
    'resetPassword.messages.resetFailed': '密码重置失败，请重试',
  },
  en: {
    // Reset Password Page
    'resetPassword.title': 'Reset Password',
    'resetPassword.subtitle': 'Please enter your new password',
    'resetPassword.newPassword': 'New Password',
    'resetPassword.newPasswordPlaceholder': 'Enter new password (at least 6 characters)',
    'resetPassword.confirmPassword': 'Confirm Password',
    'resetPassword.confirmPasswordPlaceholder': 'Re-enter new password',
    'resetPassword.submit': 'Reset Password',
    'resetPassword.submitting': 'Resetting...',
    'resetPassword.backToHome': 'Back to Home',
    'resetPassword.loginNow': 'Login Now',
    
    // Invalid Token
    'resetPassword.invalid.title': 'Invalid Link',
    'resetPassword.invalid.description': 'The password reset link is invalid or has expired. Please request a new password reset.',
    
    // Success
    'resetPassword.success.title': 'Reset Successful!',
    'resetPassword.success.description': 'Your password has been successfully reset. Redirecting to login page...',
    
    // Messages
    'resetPassword.messages.invalidLink': 'Reset link is invalid or expired',
    'resetPassword.messages.linkInvalid': 'Invalid Link',
    'resetPassword.messages.tokenInvalid': 'Invalid reset link',
    'resetPassword.messages.verifyFailed': 'Verification Failed',
    'resetPassword.messages.passwordMismatch': 'Passwords do not match',
    'resetPassword.messages.passwordTooShort': 'Password must be at least 6 characters',
    'resetPassword.messages.resetSuccess': 'Password reset successfully!',
    'resetPassword.messages.resetFailed': 'Failed to reset password, please try again',
  },
  ja: {
    // パスワードリセットページ
    'resetPassword.title': 'パスワードをリセット',
    'resetPassword.subtitle': '新しいパスワードを入力してください',
    'resetPassword.newPassword': '新しいパスワード',
    'resetPassword.newPasswordPlaceholder': '新しいパスワードを入力（6文字以上）',
    'resetPassword.confirmPassword': 'パスワードの確認',
    'resetPassword.confirmPasswordPlaceholder': '新しいパスワードを再入力',
    'resetPassword.submit': 'パスワードをリセット',
    'resetPassword.submitting': 'リセット中...',
    'resetPassword.backToHome': 'ホームに戻る',
    'resetPassword.loginNow': '今すぐログイン',
    
    // 無効なトークン
    'resetPassword.invalid.title': '無効なリンク',
    'resetPassword.invalid.description': 'パスワードリセットリンクが無効または期限切れです。新しいパスワードリセットをリクエストしてください。',
    
    // 成功
    'resetPassword.success.title': 'リセット成功！',
    'resetPassword.success.description': 'パスワードが正常にリセットされました。ログインページにリダイレクトしています...',
    
    // メッセージ
    'resetPassword.messages.invalidLink': 'リセットリンクが無効または期限切れです',
    'resetPassword.messages.linkInvalid': '無効なリンク',
    'resetPassword.messages.tokenInvalid': '無効なリセットリンク',
    'resetPassword.messages.verifyFailed': '検証に失敗しました',
    'resetPassword.messages.passwordMismatch': 'パスワードが一致しません',
    'resetPassword.messages.passwordTooShort': 'パスワードは6文字以上である必要があります',
    'resetPassword.messages.resetSuccess': 'パスワードが正常にリセットされました！',
    'resetPassword.messages.resetFailed': 'パスワードのリセットに失敗しました。もう一度お試しください',
  }
}
