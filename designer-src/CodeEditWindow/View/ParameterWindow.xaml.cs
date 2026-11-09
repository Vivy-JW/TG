using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.Infrastructure;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x0200003A RID: 58
	public partial class ParameterWindow : Window
	{
		// Token: 0x0600026F RID: 623 RVA: 0x0001371E File Offset: 0x0001191E
		public ParameterWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000270 RID: 624 RVA: 0x0001372C File Offset: 0x0001192C
		public new void Show()
		{
		}

		// Token: 0x06000271 RID: 625 RVA: 0x000137AC File Offset: 0x000119AC
		public void Show(PackageKey programName, string functionName)
		{
			ParameterGroup parameterGroup = new ParameterGroup();
			IEnumerable<string> enumerable = from ap in ResourceController.GetInstance().GetProgramInfo(programName).AddPoints
				where ap.Name == "global.variable"
				select ap.Content;
			parameterGroup.SelfDefVariable = string.Join(Environment.NewLine, enumerable.ToList<string>());
			IEnumerable<string> enumerable2 = from v in ResourceController.GetInstance().GetProgramInfo(programName).Variables
				where v.FunctionName == null
				select string.Format("DEFINE\t{0}\t{1}\t{2}", v.Name, v.Content, v.Description);
			parameterGroup.SystemModule = string.Join(Environment.NewLine, enumerable2.ToList<string>());
			IEnumerable<string> enumerable3 = from v in ResourceController.GetInstance().GetProgramInfo(programName).Variables
				where v.FunctionName == functionName
				select string.Format("DEFINE\t{0}\t{1}\t{2}", v.Name, v.Content, v.Description);
			parameterGroup.Local = string.Join(Environment.NewLine, enumerable3.ToList<string>());
			base.DataContext = parameterGroup;
			base.Owner = Application.Current.MainWindow;
			base.ShowDialog();
		}

		// Token: 0x06000272 RID: 626 RVA: 0x0001391D File Offset: 0x00011B1D
		private void Close_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(true);
			base.Close();
		}
	}
}
