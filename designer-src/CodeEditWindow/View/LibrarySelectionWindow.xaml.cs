using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Markup;
using SpecDesigner.FormDataEditor;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000030 RID: 48
	public partial class LibrarySelectionWindow : Window
	{
		// Token: 0x17000048 RID: 72
		// (get) Token: 0x060001E0 RID: 480 RVA: 0x0000F660 File Offset: 0x0000D860
		public string SelectedID
		{
			get
			{
				if (this.mdView != null)
				{
					return this.mdView.SelectedID;
				}
				return string.Empty;
			}
		}

		// Token: 0x060001E1 RID: 481 RVA: 0x0000F67B File Offset: 0x0000D87B
		public LibrarySelectionWindow()
		{
			base.AddHandler(MasterDetailView.ItemSelectedEvent, new RoutedEventHandler(this.mdView_ItemSelected));
			base.AddHandler(ToggleButton.CheckedEvent, new RoutedEventHandler(this.OnRadioButtonChecked));
			this.InitializeComponent();
		}

		// Token: 0x060001E2 RID: 482 RVA: 0x0000F6B7 File Offset: 0x0000D8B7
		private void mdView_ItemSelected(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(true);
			base.Close();
		}

		// Token: 0x060001E3 RID: 483 RVA: 0x0000F6CC File Offset: 0x0000D8CC
		private void OnRadioButtonChecked(object sender, RoutedEventArgs e)
		{
			RadioButton radioButton = e.OriginalSource as RadioButton;
			if (radioButton != null)
			{
				if (this.mainPanel.Children.Count > 1)
				{
					this.mainPanel.Children.RemoveAt(this.mainPanel.Children.Count - 1);
				}
				if (string.Equals(radioButton.Tag.ToString(), "Library"))
				{
					this.mdView = new MasterDetailView(SettingManager.Get().Info_Libraries);
				}
				else
				{
					this.mdView = new MasterDetailView(SettingManager.Get().Info_Subroutines);
				}
				this.mainPanel.Children.Add(this.mdView);
			}
		}

		// Token: 0x060001E4 RID: 484 RVA: 0x0000F77A File Offset: 0x0000D97A
		public void Dispose()
		{
		}

		// Token: 0x040000CA RID: 202
		private MasterDetailView mdView;
	}
}
