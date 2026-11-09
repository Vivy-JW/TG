using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using System.Windows.Threading;
using SpecDesignerCommon.Site.ViewModels;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x02000129 RID: 297
	public partial class LoginWindow : Window
	{
		// Token: 0x170002BC RID: 700
		// (get) Token: 0x06000A6D RID: 2669 RVA: 0x00033B04 File Offset: 0x00031D04
		public SettingModel Setting
		{
			get
			{
				return SettingManager.Get().CurrentSetting;
			}
		}

		// Token: 0x06000A6E RID: 2670 RVA: 0x00033B20 File Offset: 0x00031D20
		public LoginWindow()
		{
			this.InitializeComponent();
			base.Title = Application.Current.FindResource("Con_Login") as string;
			base.Dispatcher.BeginInvoke(DispatcherPriority.Background, new Action(delegate
			{
				Keyboard.Focus(this.loginBox);
			}));
		}

		// Token: 0x06000A6F RID: 2671 RVA: 0x00033B74 File Offset: 0x00031D74
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
					return;
				}
			}
			else if (Keyboard.Modifiers == ModifierKeys.None)
			{
				ConnectionManager.CloseTelnetChannel();
			}
		}

		// Token: 0x06000A70 RID: 2672 RVA: 0x00033BCC File Offset: 0x00031DCC
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

		// Token: 0x06000A71 RID: 2673 RVA: 0x00033C00 File Offset: 0x00031E00
		public new void Show()
		{
			base.ShowDialog();
		}

		// Token: 0x06000A72 RID: 2674 RVA: 0x00033C09 File Offset: 0x00031E09
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			this.FinishInput();
			base.Close();
		}

		// Token: 0x06000A73 RID: 2675 RVA: 0x00033C17 File Offset: 0x00031E17
		private void FinishInput()
		{
			ConnectionInfo.This.Area = this.areaBox.Text;
			SettingManager.Get().CurrentSetting.Connection.Save();
			base.DialogResult = new bool?(true);
		}

		// Token: 0x06000A74 RID: 2676 RVA: 0x00033C4E File Offset: 0x00031E4E
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
			base.Close();
		}
	}
}
