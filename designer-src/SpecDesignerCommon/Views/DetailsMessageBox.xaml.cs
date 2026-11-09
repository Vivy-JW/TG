using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;

namespace SpecDesignerCommon.Views
{
	// Token: 0x02000002 RID: 2
	public partial class DetailsMessageBox : Window
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public DetailsMessageBox()
		{
			this.InitializeComponent();
		}

		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000002 RID: 2 RVA: 0x0000205E File Offset: 0x0000025E
		// (set) Token: 0x06000003 RID: 3 RVA: 0x00002066 File Offset: 0x00000266
		public string Caption
		{
			get
			{
				return base.Title;
			}
			set
			{
				base.Title = value;
			}
		}

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x06000004 RID: 4 RVA: 0x0000206F File Offset: 0x0000026F
		// (set) Token: 0x06000005 RID: 5 RVA: 0x00002077 File Offset: 0x00000277
		public MessageBoxResult MessageBoxResult
		{
			get
			{
				return this._result;
			}
			private set
			{
				this._result = value;
				if (MessageBoxResult.Cancel == this._result)
				{
					base.DialogResult = new bool?(false);
					return;
				}
				base.DialogResult = new bool?(true);
			}
		}

		// Token: 0x17000003 RID: 3
		// (get) Token: 0x06000006 RID: 6 RVA: 0x000020A2 File Offset: 0x000002A2
		// (set) Token: 0x06000007 RID: 7 RVA: 0x000020B4 File Offset: 0x000002B4
		public MessageBoxResult DefaultResult
		{
			get
			{
				return (MessageBoxResult)base.GetValue(DetailsMessageBox.DefaultResultProperty);
			}
			set
			{
				base.SetValue(DetailsMessageBox.DefaultResultProperty, value);
				switch (value)
				{
				case MessageBoxResult.None:
				case (MessageBoxResult)3:
				case (MessageBoxResult)4:
				case (MessageBoxResult)5:
					break;
				case MessageBoxResult.OK:
					this._ok.IsDefault = true;
					return;
				case MessageBoxResult.Cancel:
					this._cancel.IsDefault = true;
					return;
				case MessageBoxResult.Yes:
					this._yes.IsDefault = true;
					break;
				case MessageBoxResult.No:
					this._no.IsDefault = true;
					return;
				default:
					return;
				}
			}
		}

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x06000008 RID: 8 RVA: 0x0000212E File Offset: 0x0000032E
		// (set) Token: 0x06000009 RID: 9 RVA: 0x00002140 File Offset: 0x00000340
		public string Message
		{
			get
			{
				return (string)base.GetValue(DetailsMessageBox.MessageProperty);
			}
			set
			{
				base.SetValue(DetailsMessageBox.MessageProperty, value);
			}
		}

		// Token: 0x17000005 RID: 5
		// (get) Token: 0x0600000A RID: 10 RVA: 0x0000214E File Offset: 0x0000034E
		// (set) Token: 0x0600000B RID: 11 RVA: 0x00002160 File Offset: 0x00000360
		public MessageBoxButton MessageBoxButton
		{
			get
			{
				return (MessageBoxButton)base.GetValue(DetailsMessageBox.MessageBoxButtonProperty);
			}
			set
			{
				base.SetValue(DetailsMessageBox.MessageBoxButtonProperty, value);
				switch (value)
				{
				case MessageBoxButton.OK:
					this._ok.Visibility = Visibility.Visible;
					return;
				case MessageBoxButton.OKCancel:
					this._ok.Visibility = Visibility.Visible;
					this._cancel.Visibility = Visibility.Visible;
					return;
				case (MessageBoxButton)2:
					break;
				case MessageBoxButton.YesNoCancel:
					this._yes.Visibility = Visibility.Visible;
					this._no.Visibility = Visibility.Visible;
					this._cancel.Visibility = Visibility.Visible;
					break;
				case MessageBoxButton.YesNo:
					this._yes.Visibility = Visibility.Visible;
					this._no.Visibility = Visibility.Visible;
					return;
				default:
					return;
				}
			}
		}

		// Token: 0x17000006 RID: 6
		// (get) Token: 0x0600000C RID: 12 RVA: 0x000021FE File Offset: 0x000003FE
		// (set) Token: 0x0600000D RID: 13 RVA: 0x00002210 File Offset: 0x00000410
		public MessageBoxImage MessageBoxImage
		{
			get
			{
				return (MessageBoxImage)base.GetValue(DetailsMessageBox.MessageBoxImageProperty);
			}
			set
			{
				base.SetValue(DetailsMessageBox.MessageBoxImageProperty, value);
			}
		}

		// Token: 0x17000007 RID: 7
		// (get) Token: 0x0600000E RID: 14 RVA: 0x00002223 File Offset: 0x00000423
		// (set) Token: 0x0600000F RID: 15 RVA: 0x00002235 File Offset: 0x00000435
		public string DetailMessage
		{
			get
			{
				return (string)base.GetValue(DetailsMessageBox.DetailMessageProperty);
			}
			set
			{
				base.SetValue(DetailsMessageBox.DetailMessageProperty, value);
			}
		}

		// Token: 0x06000010 RID: 16 RVA: 0x00002243 File Offset: 0x00000443
		public static MessageBoxResult Show(string messageBoxText)
		{
			return DetailsMessageBox.Show(null, messageBoxText, string.Empty, MessageBoxButton.OK, MessageBoxImage.None, MessageBoxResult.None);
		}

		// Token: 0x06000011 RID: 17 RVA: 0x00002254 File Offset: 0x00000454
		public static MessageBoxResult Show(string messageBoxText, string caption)
		{
			return DetailsMessageBox.Show(null, messageBoxText, caption, MessageBoxButton.OK, MessageBoxImage.None, MessageBoxResult.None);
		}

		// Token: 0x06000012 RID: 18 RVA: 0x00002261 File Offset: 0x00000461
		public static MessageBoxResult Show(Window owner, string messageBoxText)
		{
			return DetailsMessageBox.Show(owner, messageBoxText, string.Empty, MessageBoxButton.OK, MessageBoxImage.None, MessageBoxResult.None);
		}

		// Token: 0x06000013 RID: 19 RVA: 0x00002272 File Offset: 0x00000472
		public static MessageBoxResult Show(string messageBoxText, string caption, MessageBoxButton button)
		{
			return DetailsMessageBox.Show(null, messageBoxText, caption, button, MessageBoxImage.None, MessageBoxResult.None);
		}

		// Token: 0x06000014 RID: 20 RVA: 0x0000227F File Offset: 0x0000047F
		public static MessageBoxResult Show(Window owner, string messageBoxText, string caption)
		{
			return DetailsMessageBox.Show(owner, messageBoxText, caption, MessageBoxButton.OK, MessageBoxImage.None, MessageBoxResult.None);
		}

		// Token: 0x06000015 RID: 21 RVA: 0x0000228C File Offset: 0x0000048C
		public static MessageBoxResult Show(string messageBoxText, string caption, MessageBoxButton button, MessageBoxImage icon)
		{
			return DetailsMessageBox.Show(null, messageBoxText, caption, button, icon, MessageBoxResult.None);
		}

		// Token: 0x06000016 RID: 22 RVA: 0x00002299 File Offset: 0x00000499
		public static MessageBoxResult Show(Window owner, string messageBoxText, string caption, MessageBoxButton button)
		{
			return DetailsMessageBox.Show(owner, messageBoxText, caption, button, MessageBoxImage.None, MessageBoxResult.None);
		}

		// Token: 0x06000017 RID: 23 RVA: 0x000022A6 File Offset: 0x000004A6
		public static MessageBoxResult Show(string messageBoxText, string caption, MessageBoxButton button, MessageBoxImage image, MessageBoxResult defaultResult)
		{
			return DetailsMessageBox.Show(null, messageBoxText, caption, button, image, defaultResult);
		}

		// Token: 0x06000018 RID: 24 RVA: 0x000022B4 File Offset: 0x000004B4
		public static MessageBoxResult Show(Window owner, string messageBoxText, string caption, MessageBoxButton button, MessageBoxImage icon)
		{
			return DetailsMessageBox.Show(owner, messageBoxText, caption, button, icon, MessageBoxResult.None);
		}

		// Token: 0x06000019 RID: 25 RVA: 0x000022C2 File Offset: 0x000004C2
		public static MessageBoxResult Show(Window owner, string messageBoxText, string caption, MessageBoxButton button, MessageBoxImage icon, MessageBoxResult defaultResult)
		{
			return DetailsMessageBox.Show(owner, messageBoxText, string.Empty, caption, button, icon, MessageBoxResult.None);
		}

		// Token: 0x0600001A RID: 26 RVA: 0x000022D5 File Offset: 0x000004D5
		public static MessageBoxResult Show(string messageBoxText, string detailmessageBoxText, string caption, MessageBoxButton button, MessageBoxImage icon)
		{
			return DetailsMessageBox.Show(null, messageBoxText, detailmessageBoxText, caption, button, icon, MessageBoxResult.None);
		}

		// Token: 0x0600001B RID: 27 RVA: 0x000022E4 File Offset: 0x000004E4
		public static MessageBoxResult Show(Window owner, string messageBoxText, string detailmessageBoxText, string caption, MessageBoxButton button, MessageBoxImage icon, MessageBoxResult defaultResult)
		{
			DetailsMessageBox detailsMessageBox = new DetailsMessageBox();
			detailsMessageBox.Caption = caption;
			detailsMessageBox.DefaultResult = defaultResult;
			detailsMessageBox.Owner = owner;
			detailsMessageBox.Message = messageBoxText;
			detailsMessageBox.DetailMessage = detailmessageBoxText;
			detailsMessageBox.MessageBoxButton = button;
			detailsMessageBox.MessageBoxImage = icon;
			if (detailsMessageBox.ShowDialog() == false)
			{
				return MessageBoxResult.Cancel;
			}
			return detailsMessageBox.MessageBoxResult;
		}

		// Token: 0x0600001C RID: 28 RVA: 0x0000234F File Offset: 0x0000054F
		private void cancel_Click(object sender, RoutedEventArgs e)
		{
			this.MessageBoxResult = MessageBoxResult.Cancel;
		}

		// Token: 0x0600001D RID: 29 RVA: 0x00002358 File Offset: 0x00000558
		private void no_Click(object sender, RoutedEventArgs e)
		{
			this.MessageBoxResult = MessageBoxResult.No;
		}

		// Token: 0x0600001E RID: 30 RVA: 0x00002361 File Offset: 0x00000561
		private void ok_Click(object sender, RoutedEventArgs e)
		{
			this.MessageBoxResult = MessageBoxResult.OK;
		}

		// Token: 0x0600001F RID: 31 RVA: 0x0000236A File Offset: 0x0000056A
		private void this_Loaded(object sender, RoutedEventArgs e)
		{
			this._close = (Button)base.Template.FindName("PART_Close", this);
			if (this._close != null && !this._cancel.IsVisible)
			{
				this._close.IsCancel = false;
			}
		}

		// Token: 0x06000020 RID: 32 RVA: 0x000023A9 File Offset: 0x000005A9
		private void yes_Click(object sender, RoutedEventArgs e)
		{
			this.MessageBoxResult = MessageBoxResult.Yes;
		}

		// Token: 0x06000021 RID: 33 RVA: 0x000023B4 File Offset: 0x000005B4
		private void MessageTextBox_MouseUp(object sender, MouseButtonEventArgs e)
		{
			TextBox textBox = (TextBox)sender;
			if (textBox != null)
			{
				textBox.SelectAll();
			}
		}

		// Token: 0x06000024 RID: 36 RVA: 0x00002504 File Offset: 0x00000704
		[EditorBrowsable(EditorBrowsableState.Never)]
		[DebuggerNonUserCode]
		[GeneratedCode("PresentationBuildTasks", "4.0.0.0")]
		void IStyleConnector.Connect(int connectionId, object target)
		{
			if (connectionId != 2)
			{
				return;
			}
			((TextBox)target).MouseLeftButtonUp += this.MessageTextBox_MouseUp;
		}

		// Token: 0x04000001 RID: 1
		private MessageBoxResult _result;

		// Token: 0x04000002 RID: 2
		private Button _close;

		// Token: 0x04000003 RID: 3
		public static readonly DependencyProperty DefaultResultProperty = DependencyProperty.Register("DefaultResult", typeof(MessageBoxResult), typeof(DetailsMessageBox), new UIPropertyMetadata(MessageBoxResult.None));

		// Token: 0x04000004 RID: 4
		public static readonly DependencyProperty MessageProperty = DependencyProperty.Register("Message", typeof(string), typeof(DetailsMessageBox), new UIPropertyMetadata(string.Empty));

		// Token: 0x04000005 RID: 5
		public static readonly DependencyProperty MessageBoxButtonProperty = DependencyProperty.Register("MessageBoxButton", typeof(MessageBoxButton), typeof(DetailsMessageBox), new UIPropertyMetadata(MessageBoxButton.OK));

		// Token: 0x04000006 RID: 6
		public static readonly DependencyProperty MessageBoxImageProperty = DependencyProperty.Register("MessageBoxImage", typeof(MessageBoxImage), typeof(DetailsMessageBox), new UIPropertyMetadata(MessageBoxImage.None));

		// Token: 0x04000007 RID: 7
		public static readonly DependencyProperty DetailMessageProperty = DependencyProperty.Register("DetailMessage", typeof(string), typeof(DetailsMessageBox), new PropertyMetadata(string.Empty));
	}
}
