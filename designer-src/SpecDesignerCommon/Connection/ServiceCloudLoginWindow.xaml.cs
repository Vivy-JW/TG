using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Net;
using System.Text;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using Newtonsoft.Json;
using SpecDesignerPreference;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x0200008F RID: 143
	public partial class ServiceCloudLoginWindow : Window
	{
		// Token: 0x060005D9 RID: 1497 RVA: 0x0001AC70 File Offset: 0x00018E70
		public ServiceCloudLoginWindow()
		{
			this.InitializeComponent();
			this.element = new PreferenceModel(PreferenceManager.Current.Settings.ToString());
			base.DataContext = this.element;
			base.Title = Application.Current.FindResource("menu_ServiceCloudSetting") as string;
		}

		// Token: 0x060005DA RID: 1498 RVA: 0x0001ACC9 File Offset: 0x00018EC9
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

		// Token: 0x060005DB RID: 1499 RVA: 0x0001AD08 File Offset: 0x00018F08
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

		// Token: 0x060005DC RID: 1500 RVA: 0x0001AD3C File Offset: 0x00018F3C
		public new void Show()
		{
			base.ShowDialog();
		}

		// Token: 0x060005DD RID: 1501 RVA: 0x0001AD45 File Offset: 0x00018F45
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			PreferenceManager.Current.Save(this.element);
			this.FinishInput();
			base.Close();
		}

		// Token: 0x060005DE RID: 1502 RVA: 0x0001AD63 File Offset: 0x00018F63
		private void FinishInput()
		{
			base.DialogResult = new bool?(true);
		}

		// Token: 0x060005DF RID: 1503 RVA: 0x0001AD71 File Offset: 0x00018F71
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
			base.Close();
		}

		// Token: 0x060005E0 RID: 1504 RVA: 0x0001AD88 File Offset: 0x00018F88
		private void ConnectionTest_Cick(object sender, RoutedEventArgs e)
		{
			string text = string.Empty;
			try
			{
				string text2 = "http://tscloud.digiwin.com/DWGateway/restful/TCPortal/ILoginService/login";
				string text3 = string.Concat(new string[]
				{
					"{\"user\":{\"account\":\"",
					this.element.ServiceCloudLogin,
					"\",\"password\":\"",
					this.element.ServiceCloudPassword,
					"\"}}"
				});
				byte[] bytes = Encoding.UTF8.GetBytes(text3);
				HttpWebRequest httpWebRequest = (HttpWebRequest)WebRequest.Create(text2);
				httpWebRequest.ContentType = "application/json";
				httpWebRequest.Method = "POST";
				httpWebRequest.ContentLength = (long)bytes.Length;
				using (Stream requestStream = httpWebRequest.GetRequestStream())
				{
					requestStream.Write(bytes, 0, bytes.Length);
					requestStream.Flush();
				}
				using (HttpWebResponse httpWebResponse = (HttpWebResponse)httpWebRequest.GetResponse())
				{
					using (Stream responseStream = httpWebResponse.GetResponseStream())
					{
						using (StreamReader streamReader = new StreamReader(responseStream))
						{
							string text4 = streamReader.ReadToEnd();
							ServiceCloudLoginWindow.cloudmsg cloudmsg = JsonConvert.DeserializeObject<ServiceCloudLoginWindow.cloudmsg>(text4);
							if (cloudmsg.statusDescription == "OK")
							{
								MessageBox.Show(Application.Current.FindResource("Message_ConnectionOK") as string);
							}
						}
					}
				}
			}
			catch (WebException ex)
			{
				if (ex.Status == WebExceptionStatus.ProtocolError)
				{
					HttpWebResponse httpWebResponse2 = ex.Response as HttpWebResponse;
					if (httpWebResponse2 != null)
					{
						string text5 = new StreamReader(httpWebResponse2.GetResponseStream()).ReadToEnd();
						ServiceCloudLoginWindow.cloudmsg cloudmsg2 = JsonConvert.DeserializeObject<ServiceCloudLoginWindow.cloudmsg>(text5);
						text = string.Format("{0}", cloudmsg2.errorMessage);
						MessageBox.Show(text);
					}
				}
			}
		}

		// Token: 0x060005E1 RID: 1505 RVA: 0x0001AFAC File Offset: 0x000191AC
		private void ClearButton_Cick(object sender, RoutedEventArgs e)
		{
			this.account.Text = "";
			this.password.Password = "";
		}

		// Token: 0x0400023C RID: 572
		private PreferenceModel element;

		// Token: 0x02000090 RID: 144
		public class cloudmsg
		{
			// Token: 0x170001AE RID: 430
			// (get) Token: 0x060005E5 RID: 1509 RVA: 0x0001B0E9 File Offset: 0x000192E9
			// (set) Token: 0x060005E6 RID: 1510 RVA: 0x0001B0F1 File Offset: 0x000192F1
			public string errorMessage { get; set; }

			// Token: 0x170001AF RID: 431
			// (get) Token: 0x060005E7 RID: 1511 RVA: 0x0001B0FA File Offset: 0x000192FA
			// (set) Token: 0x060005E8 RID: 1512 RVA: 0x0001B102 File Offset: 0x00019302
			public string statusDescription { get; set; }
		}
	}
}
