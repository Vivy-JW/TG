using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesignerCommon.Site.Views
{
	// Token: 0x02000114 RID: 276
	public static class PasswordHelper
	{
		// Token: 0x060009DD RID: 2525 RVA: 0x00031424 File Offset: 0x0002F624
		public static void SetAttach(DependencyObject dp, bool value)
		{
			dp.SetValue(PasswordHelper.AttachProperty, value);
		}

		// Token: 0x060009DE RID: 2526 RVA: 0x00031437 File Offset: 0x0002F637
		public static bool GetAttach(DependencyObject dp)
		{
			return (bool)dp.GetValue(PasswordHelper.AttachProperty);
		}

		// Token: 0x060009DF RID: 2527 RVA: 0x00031449 File Offset: 0x0002F649
		public static string GetPassword(DependencyObject dp)
		{
			return (string)dp.GetValue(PasswordHelper.PasswordProperty);
		}

		// Token: 0x060009E0 RID: 2528 RVA: 0x0003145B File Offset: 0x0002F65B
		public static void SetPassword(DependencyObject dp, string value)
		{
			dp.SetValue(PasswordHelper.PasswordProperty, value);
		}

		// Token: 0x060009E1 RID: 2529 RVA: 0x00031469 File Offset: 0x0002F669
		private static bool GetIsUpdating(DependencyObject dp)
		{
			return (bool)dp.GetValue(PasswordHelper.IsUpdatingProperty);
		}

		// Token: 0x060009E2 RID: 2530 RVA: 0x0003147B File Offset: 0x0002F67B
		private static void SetIsUpdating(DependencyObject dp, bool value)
		{
			dp.SetValue(PasswordHelper.IsUpdatingProperty, value);
		}

		// Token: 0x060009E3 RID: 2531 RVA: 0x00031490 File Offset: 0x0002F690
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

		// Token: 0x060009E4 RID: 2532 RVA: 0x000314E4 File Offset: 0x0002F6E4
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

		// Token: 0x060009E5 RID: 2533 RVA: 0x0003153C File Offset: 0x0002F73C
		private static void PasswordChanged(object sender, RoutedEventArgs e)
		{
			PasswordBox passwordBox = sender as PasswordBox;
			PasswordHelper.SetIsUpdating(passwordBox, true);
			PasswordHelper.SetPassword(passwordBox, passwordBox.Password);
			PasswordHelper.SetIsUpdating(passwordBox, false);
		}

		// Token: 0x040003BA RID: 954
		public static readonly DependencyProperty PasswordProperty = DependencyProperty.RegisterAttached("Password", typeof(string), typeof(PasswordHelper), new FrameworkPropertyMetadata(string.Empty, new PropertyChangedCallback(PasswordHelper.OnPasswordPropertyChanged)));

		// Token: 0x040003BB RID: 955
		public static readonly DependencyProperty AttachProperty = DependencyProperty.RegisterAttached("Attach", typeof(bool), typeof(PasswordHelper), new PropertyMetadata(false, new PropertyChangedCallback(PasswordHelper.Attach)));

		// Token: 0x040003BC RID: 956
		private static readonly DependencyProperty IsUpdatingProperty = DependencyProperty.RegisterAttached("IsUpdating", typeof(bool), typeof(PasswordHelper));
	}
}
