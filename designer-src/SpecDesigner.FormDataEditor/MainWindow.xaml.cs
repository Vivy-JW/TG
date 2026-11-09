using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesignerCommon;

namespace SpecDesigner.FormDataEditor
{
	// Token: 0x0200000A RID: 10
	public partial class MainWindow : Window
	{
		// Token: 0x06000036 RID: 54 RVA: 0x00002E49 File Offset: 0x00001049
		public MainWindow()
		{
			this.InitializeComponent();
			SettingManager.Get().LoadCommonData();
		}
	}
}
