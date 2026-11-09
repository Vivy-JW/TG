using System;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;

namespace SpecDesigner.SpecEditor
{
	// Token: 0x02000008 RID: 8
	public class SpecPropertyCommands
	{
		// Token: 0x17000006 RID: 6
		// (get) Token: 0x0600004F RID: 79 RVA: 0x00005900 File Offset: 0x00003B00
		public static RoutedCommand ShowZoomsWindowCommand
		{
			get
			{
				if (SpecPropertyCommands._showZoomsWindowCommand == null)
				{
					SpecPropertyCommands._showZoomsWindowCommand = new RoutedCommand("ShowZoomsWindowCommand", typeof(SpecPropertyCommands));
				}
				return SpecPropertyCommands._showZoomsWindowCommand;
			}
		}

		// Token: 0x06000050 RID: 80 RVA: 0x00005927 File Offset: 0x00003B27
		public static void CanShowZoomsWindow(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000051 RID: 81 RVA: 0x00005930 File Offset: 0x00003B30
		public static void ExecutedShowZoomsWindow(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			TextBox textBox = e.Parameter as TextBox;
			if (textBox != null)
			{
				SpecPropertyEditor.ShowZoomsWindow(textBox);
			}
		}

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x06000052 RID: 82 RVA: 0x00005959 File Offset: 0x00003B59
		public static RoutedCommand ClearContentCommand
		{
			get
			{
				if (SpecPropertyCommands._clearContentCommand == null)
				{
					SpecPropertyCommands._clearContentCommand = new RoutedCommand("ClearContentCommand", typeof(SpecPropertyCommands));
				}
				return SpecPropertyCommands._clearContentCommand;
			}
		}

		// Token: 0x06000053 RID: 83 RVA: 0x00005980 File Offset: 0x00003B80
		public static void CanClearContent(object sender, CanExecuteRoutedEventArgs e)
		{
			e.CanExecute = true;
		}

		// Token: 0x06000054 RID: 84 RVA: 0x0000598C File Offset: 0x00003B8C
		public static void ExecutedClearContent(object sender, ExecutedRoutedEventArgs e)
		{
			e.Handled = true;
			if (e.Parameter is TextBox)
			{
				TextBox textBox = e.Parameter as TextBox;
				textBox.Text = SpecPropertyCommands.NOSET;
				BindingExpression bindingExpression = textBox.GetBindingExpression(TextBox.TextProperty);
				bindingExpression.UpdateSource();
				return;
			}
			if (e.Parameter is ComboBox)
			{
				ComboBox comboBox = e.Parameter as ComboBox;
				comboBox.Text = SpecPropertyCommands.NOSET;
				BindingExpression bindingExpression2 = comboBox.GetBindingExpression(ComboBox.TextProperty);
				bindingExpression2.UpdateSource();
			}
		}

		// Token: 0x17000008 RID: 8
		// (get) Token: 0x06000055 RID: 85 RVA: 0x00005A0D File Offset: 0x00003C0D
		public static RoutedCommand ShowItemsWindowCommand
		{
			get
			{
				if (SpecPropertyCommands._showItemsWindowCommand == null)
				{
					SpecPropertyCommands._showItemsWindowCommand = new RoutedCommand("ShowItemsWindowCommand", typeof(SpecPropertyCommands));
				}
				return SpecPropertyCommands._showItemsWindowCommand;
			}
		}

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000056 RID: 86 RVA: 0x00005A34 File Offset: 0x00003C34
		public static RoutedCommand ShowLocalStringWindowCommand
		{
			get
			{
				if (SpecPropertyCommands._showLocalStringWindowCommand == null)
				{
					SpecPropertyCommands._showLocalStringWindowCommand = new RoutedCommand("ShowLocalStringWindowCommand", typeof(SpecPropertyCommands));
				}
				return SpecPropertyCommands._showLocalStringWindowCommand;
			}
		}

		// Token: 0x04000043 RID: 67
		private static RoutedCommand _showZoomsWindowCommand;

		// Token: 0x04000044 RID: 68
		private static RoutedCommand _clearContentCommand;

		// Token: 0x04000045 RID: 69
		private static readonly string NOSET = "[[RE][SET]]";

		// Token: 0x04000046 RID: 70
		private static RoutedCommand _showItemsWindowCommand;

		// Token: 0x04000047 RID: 71
		private static RoutedCommand _showLocalStringWindowCommand;
	}
}
