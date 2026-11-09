using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesignerCommon.Connection.Helper
{
	// Token: 0x0200012A RID: 298
	public static class PasswordHelper
	{
		// Token: 0x06000A79 RID: 2681 RVA: 0x00033D95 File Offset: 0x00031F95
		public static void SetAttach(DependencyObject dp, bool value)
		{
			dp.SetValue(PasswordHelper.AttachProperty, value);
		}

		// Token: 0x06000A7A RID: 2682 RVA: 0x00033DA8 File Offset: 0x00031FA8
		public static bool GetAttach(DependencyObject dp)
		{
			return (bool)dp.GetValue(PasswordHelper.AttachProperty);
		}

		// Token: 0x06000A7B RID: 2683 RVA: 0x00033DBA File Offset: 0x00031FBA
		public static string GetPassword(DependencyObject dp)
		{
			return (string)dp.GetValue(PasswordHelper.PasswordProperty);
		}

		// Token: 0x06000A7C RID: 2684 RVA: 0x00033DCC File Offset: 0x00031FCC
		public static void SetPassword(DependencyObject dp, string value)
		{
			dp.SetValue(PasswordHelper.PasswordProperty, value);
		}

		// Token: 0x06000A7D RID: 2685 RVA: 0x00033DDA File Offset: 0x00031FDA
		private static bool GetIsUpdating(DependencyObject dp)
		{
			return (bool)dp.GetValue(PasswordHelper.IsUpdatingProperty);
		}

		// Token: 0x06000A7E RID: 2686 RVA: 0x00033DEC File Offset: 0x00031FEC
		private static void SetIsUpdating(DependencyObject dp, bool value)
		{
			dp.SetValue(PasswordHelper.IsUpdatingProperty, value);
		}

		// Token: 0x06000A7F RID: 2687 RVA: 0x00033E00 File Offset: 0x00032000
		private static void OnPasswordPropertyChanged(DependencyObject sender, DependencyPropertyChangedEventArgs e)
		{
			PasswordBox passwordBox = sender as PasswordBox;
			passwordBox.PasswordChanged -= PasswordHelper.PasswordChanged;
			if (!PasswordHelper.GetIsUpdating(passwordBox))
			{
				passwordBox.Password = (string)e.NewValue;
			}
			passwordBox.PasswordChanged += PasswordHelper.PasswordChanged;
		}

		// Token: 0x06000A80 RID: 2688 RVA: 0x00033E54 File Offset: 0x00032054
		private static void Attach(DependencyObject sender, DependencyPropertyChangedEventArgs e)
		{
			PasswordBox passwordBox = sender as PasswordBox;
			if (passwordBox == null)
			{
				return;
			}
			if ((bool)e.OldValue)
			{
				passwordBox.PasswordChanged -= PasswordHelper.PasswordChanged;
			}
			if ((bool)e.NewValue)
			{
				passwordBox.PasswordChanged += PasswordHelper.PasswordChanged;
			}
		}

		// Token: 0x06000A81 RID: 2689 RVA: 0x00033EAC File Offset: 0x000320AC
		private static void PasswordChanged(object sender, RoutedEventArgs e)
		{
			PasswordBox passwordBox = sender as PasswordBox;
			PasswordHelper.SetIsUpdating(passwordBox, true);
			PasswordHelper.SetPassword(passwordBox, passwordBox.Password);
			PasswordHelper.SetIsUpdating(passwordBox, false);
		}

		// Token: 0x040003F5 RID: 1013
		public static readonly DependencyProperty PasswordProperty = DependencyProperty.RegisterAttached("Password", typeof(string), typeof(PasswordHelper), new FrameworkPropertyMetadata(string.Empty, new PropertyChangedCallback(PasswordHelper.OnPasswordPropertyChanged)));

		// Token: 0x040003F6 RID: 1014
		public static readonly DependencyProperty AttachProperty = DependencyProperty.RegisterAttached("Attach", typeof(bool), typeof(PasswordHelper), new PropertyMetadata(false, new PropertyChangedCallback(PasswordHelper.Attach)));

		// Token: 0x040003F7 RID: 1015
		private static readonly DependencyProperty IsUpdatingProperty = DependencyProperty.RegisterAttached("IsUpdating", typeof(bool), typeof(PasswordHelper));
	}
}
