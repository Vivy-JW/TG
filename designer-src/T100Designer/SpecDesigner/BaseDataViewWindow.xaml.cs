using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Markup;
using System.Windows.Media;
using SpecDesigner.FormDataEditor;
using SpecDesignerCommon;

namespace SpecDesigner
{
	// Token: 0x02000031 RID: 49
	public partial class BaseDataViewWindow : Window
	{
		// Token: 0x06000286 RID: 646 RVA: 0x0000BBE0 File Offset: 0x00009DE0
		public BaseDataViewWindow()
		{
			this.InitializeComponent();
			base.AddHandler(ButtonBase.ClickEvent, new RoutedEventHandler(this.button_Click));
		}

		// Token: 0x06000287 RID: 647 RVA: 0x0000BC08 File Offset: 0x00009E08
		private void button_Click(object sender, RoutedEventArgs e)
		{
			if (this.lastClickButton != null)
			{
				this.lastClickButton.Foreground = Brushes.Black;
				this.lastClickButton.FontWeight = FontWeights.Normal;
			}
			Button button = e.Source as Button;
			if (button != null)
			{
				this.lastClickButton = button;
				this.lastClickButton.Foreground = Brushes.Blue;
				this.lastClickButton.FontWeight = FontWeights.Bold;
			}
		}

		// Token: 0x06000288 RID: 648 RVA: 0x0000BC73 File Offset: 0x00009E73
		private void itemsButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_Items));
		}

		// Token: 0x06000289 RID: 649 RVA: 0x0000BCA5 File Offset: 0x00009EA5
		private void messageButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_Messages));
		}

		// Token: 0x0600028A RID: 650 RVA: 0x0000BCD7 File Offset: 0x00009ED7
		private void zoomsButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_Zooms));
		}

		// Token: 0x0600028B RID: 651 RVA: 0x0000BD09 File Offset: 0x00009F09
		private void checksButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_Checks));
		}

		// Token: 0x0600028C RID: 652 RVA: 0x0000BD3B File Offset: 0x00009F3B
		private void progrelButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_ProgRel));
		}

		// Token: 0x0600028D RID: 653 RVA: 0x0000BD6D File Offset: 0x00009F6D
		private void subroutinesButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_Subroutines));
		}

		// Token: 0x0600028E RID: 654 RVA: 0x0000BD9F File Offset: 0x00009F9F
		private void librariesButton_Click(object sender, RoutedEventArgs e)
		{
			this.dataViewContainer.Children.Clear();
			this.dataViewContainer.Children.Add(new MasterDetailView(SettingManager.Get().Info_Libraries));
		}

		// Token: 0x04000171 RID: 369
		private Button lastClickButton;
	}
}
