using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure;
using SpecDesignerCommon;

namespace SpecDesigner
{
	// Token: 0x02000035 RID: 53
	public partial class CodeTemplateSelector : Window, INotifyPropertyChanged
	{
		// Token: 0x060002A9 RID: 681 RVA: 0x0000C514 File Offset: 0x0000A714
		public CodeTemplateSelector(PackageKey programKey)
		{
			this.InitializeComponent();
			this._programKey = programKey;
			if (null == this._programKey)
			{
				return;
			}
			base.Title = string.Format("{0} [{1}]", Application.Current.FindResource("menu_changeProgramTemplate") as string, this._programKey.Program);
			this.CTSelectorRoot.DataContext = this;
			this.CodeTB.SelectedValue = ResourceController.GetInstance().GetProgramInfo(programKey).GetCodeTemplate();
		}

		// Token: 0x170000A4 RID: 164
		// (get) Token: 0x060002AA RID: 682 RVA: 0x0000C5F0 File Offset: 0x0000A7F0
		public ObservableCollection<CodeTemplate> CodeTemplates
		{
			get
			{
				if (this._templates.Count == 0)
				{
					CodeTemplateComparer codeTemplateComparer = new CodeTemplateComparer();
					string codeTemplate = ResourceController.GetInstance().GetProgramInfo(this._programKey).GetCodeTemplate();
					string text = Path.Combine(SettingManager.Get().GetWorkspacePath(), "mta/code_template.xml");
					if (File.Exists(text))
					{
						XElement xelement = XElement.Load(text, LoadOptions.None);
						using (IEnumerator<XElement> enumerator = xelement.Elements("kind").GetEnumerator())
						{
							while (enumerator.MoveNext())
							{
								XElement xelement2 = enumerator.Current;
								if (this.codeGroup1.Contains(codeTemplate) && this.codeGroup1.Contains(xelement2.Attribute("id").Value))
								{
									this._templates.Add(new CodeTemplate
									{
										Code = xelement2.Attribute("id").Value,
										Text = string.Format("{0}.{1}", xelement2.Attribute("id").Value, xelement2.Attribute("desc").Value)
									});
								}
								if (this.codeGroup2.Contains(codeTemplate) && this.codeGroup2.Contains(xelement2.Attribute("id").Value))
								{
									CodeTemplate codeTemplate2 = new CodeTemplate
									{
										Code = xelement2.Attribute("id").Value,
										Text = string.Format("{0}.{1}", xelement2.Attribute("id").Value, xelement2.Attribute("desc").Value)
									};
									if (!this._templates.Contains(codeTemplate2, codeTemplateComparer))
									{
										this._templates.Add(codeTemplate2);
									}
								}
							}
							goto IL_020A;
						}
					}
					DesignerMessageBox.Show(Application.Current.FindResource("Message_CodeTemplateFileNotFound") as string);
				}
				IL_020A:
				return this._templates;
			}
		}

		// Token: 0x060002AB RID: 683 RVA: 0x0000C82C File Offset: 0x0000AA2C
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x14000007 RID: 7
		// (add) Token: 0x060002AC RID: 684 RVA: 0x0000C848 File Offset: 0x0000AA48
		// (remove) Token: 0x060002AD RID: 685 RVA: 0x0000C880 File Offset: 0x0000AA80
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060002AE RID: 686 RVA: 0x0000C8B8 File Offset: 0x0000AAB8
		private void acceptButton_Click(object sender, RoutedEventArgs e)
		{
			CodeTemplate codeTemplate = (CodeTemplate)this.CodeTB.SelectedItem;
			ResourceController.GetInstance().GetProgramInfo(this._programKey).SetCodeTemplate(codeTemplate.Code);
			base.Close();
		}

		// Token: 0x060002AF RID: 687 RVA: 0x0000C8F8 File Offset: 0x0000AAF8
		private void cancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.Close();
		}

		// Token: 0x04000180 RID: 384
		private PackageKey _programKey;

		// Token: 0x04000181 RID: 385
		private string[] codeGroup1 = new string[] { "F", "Q", "W" };

		// Token: 0x04000182 RID: 386
		private string[] codeGroup2 = new string[] { "P", "R", "W" };

		// Token: 0x04000183 RID: 387
		public ObservableCollection<CodeTemplate> _templates = new ObservableCollection<CodeTemplate>();
	}
}
