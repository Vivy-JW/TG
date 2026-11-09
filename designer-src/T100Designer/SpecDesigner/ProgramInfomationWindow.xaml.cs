using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner
{
	// Token: 0x02000004 RID: 4
	public partial class ProgramInfomationWindow : Window
	{
		// Token: 0x06000039 RID: 57 RVA: 0x00002A75 File Offset: 0x00000C75
		public ProgramInfomationWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x0600003A RID: 58 RVA: 0x00002A83 File Offset: 0x00000C83
		public void Load(PackageKey key, XElement source)
		{
			this.Key = key;
			base.DataContext = source;
			this.ShowColumn();
		}

		// Token: 0x0600003B RID: 59 RVA: 0x00002A9C File Offset: 0x00000C9C
		private void ShowColumn()
		{
			TzpManager tzpManger = SettingManager.Get().GetTzpManger(this.Key);
			if (tzpManger.IsStandardProgram)
			{
				this.stdprogTitle.Visibility = Visibility.Collapsed;
				this.stdporgValue.Visibility = Visibility.Collapsed;
			}
			if (tzpManger.Type != TzpType.Code && tzpManger.Type != TzpType.ReportCode)
			{
				this.sectionTitle.Visibility = Visibility.Collapsed;
				this.sectionValue.Visibility = Visibility.Collapsed;
				this.templateTitle.Visibility = Visibility.Collapsed;
				this.templateValue.Visibility = Visibility.Collapsed;
			}
			switch (tzpManger.Type)
			{
			case TzpType.ReportSpec:
			case TzpType.CodeSpec:
				this.typeTitle.Visibility = Visibility.Collapsed;
				this.typeValue.Visibility = Visibility.Collapsed;
				this.classTitle.Visibility = Visibility.Collapsed;
				this.classValue.Visibility = Visibility.Collapsed;
				base.Title = Application.Current.FindResource("menu_AboutSpec") as string;
				return;
			case TzpType.ReportCode:
			case TzpType.Code:
				this.classTitle.Visibility = Visibility.Collapsed;
				this.classValue.Visibility = Visibility.Collapsed;
				base.Title = Application.Current.FindResource("menu_AboutProgram") as string;
				return;
			case TzpType.Form:
				base.Title = Application.Current.FindResource("menu_AboutSpec") as string;
				return;
			default:
				base.Title = Application.Current.FindResource("menu_AboutProgram") as string;
				return;
			}
		}

		// Token: 0x04000014 RID: 20
		private PackageKey Key;
	}
}
