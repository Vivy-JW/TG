using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesignerCommon.Site.ViewModels;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200008E RID: 142
	public partial class TOPSTDLoginWindow : Window
	{
		// Token: 0x170001AD RID: 429
		// (get) Token: 0x060005CC RID: 1484 RVA: 0x0001AA3E File Offset: 0x00018C3E
		// (set) Token: 0x060005CD RID: 1485 RVA: 0x0001AA46 File Offset: 0x00018C46
		public TOPSTDSettingModel Setting { get; private set; }

		// Token: 0x060005CE RID: 1486 RVA: 0x0001AA4F File Offset: 0x00018C4F
		public TOPSTDLoginWindow()
		{
			this.InitializeComponent();
			base.Title = Application.Current.FindResource("Con_Login") as string;
		}

		// Token: 0x060005CF RID: 1487 RVA: 0x0001AA78 File Offset: 0x00018C78
		public TOPSTDLoginWindow(TOPSTDSettingModel settingModel)
			: this()
		{
			this.Setting = settingModel;
			this.promptTitle.Text = (this.Setting.IsLogin ? (Application.Current.FindResource("Connect_topstdLoggedIn") as string) : (Application.Current.FindResource("Connect_topstdUnloggedIn") as string));
			base.DataContext = this.Setting;
		}

		// Token: 0x060005D0 RID: 1488 RVA: 0x0001AAE0 File Offset: 0x00018CE0
		private void InputBox_KeyDown(object sender, KeyEventArgs e)
		{
			if (!base.IsVisible)
			{
				return;
			}
			if (e.Key == Key.Return)
			{
				if (sender is PasswordBox)
				{
					this.FinishInput();
				}
				if (sender is TextBox && !(sender as TextBox).AcceptsReturn)
				{
					this.MoveToNextUIElement(e);
				}
			}
		}

		// Token: 0x060005D1 RID: 1489 RVA: 0x0001AB20 File Offset: 0x00018D20
		private void MoveToNextUIElement(KeyEventArgs e)
		{
			FocusNavigationDirection focusNavigationDirection = FocusNavigationDirection.Next;
			TraversalRequest traversalRequest = new TraversalRequest(focusNavigationDirection);
			UIElement uielement = Keyboard.FocusedElement as UIElement;
			if (uielement != null && uielement.MoveFocus(traversalRequest))
			{
				e.Handled = true;
			}
		}

		// Token: 0x060005D2 RID: 1490 RVA: 0x0001AB54 File Offset: 0x00018D54
		public new void Show()
		{
			base.ShowDialog();
		}

		// Token: 0x060005D3 RID: 1491 RVA: 0x0001AB5D File Offset: 0x00018D5D
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			this.FinishInput();
			base.Close();
		}

		// Token: 0x060005D4 RID: 1492 RVA: 0x0001AB6B File Offset: 0x00018D6B
		private void FinishInput()
		{
			base.DialogResult = new bool?(true);
		}

		// Token: 0x060005D5 RID: 1493 RVA: 0x0001AB79 File Offset: 0x00018D79
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
			base.Close();
		}
	}
}
