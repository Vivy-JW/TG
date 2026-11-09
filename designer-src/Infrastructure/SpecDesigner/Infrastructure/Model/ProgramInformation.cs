using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using CodeEditor.FglAnalysis;
using CodeEditor.Infrastructure;
using SpecDesigner.Controls.Controls;
using SpecDesigner.Infrastructure.Event;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000043 RID: 67
	public class ProgramInformation : INotifyPropertyChanged
	{
		// Token: 0x060001C0 RID: 448 RVA: 0x00008444 File Offset: 0x00006644
		public ProgramInformation(PackageKey key)
		{
			this.ProgramKey = key;
			this._bgWorker = new BackgroundWorker();
			this._bgWorker.WorkerSupportsCancellation = true;
			this.IsTopstdMode = false;
			this.VerifyAdjustFunctionSort = false;
			this.OnPropertyChanged("AllowChangeToTopstd");
		}

		// Token: 0x060001C1 RID: 449 RVA: 0x0000853C File Offset: 0x0000673C
		public void Remove(Guid id)
		{
			AddPointModel addPointModel = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.ID == id && (a.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel == null)
			{
				throw new Exception("Can't remove" + id.ToString());
			}
			this.Remove(addPointModel.Name);
		}

		// Token: 0x060001C2 RID: 450 RVA: 0x00008634 File Offset: 0x00006834
		public void Remove(string nameToDel)
		{
			IEnumerable<AddPointModel> enumerable = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == nameToDel && (a.Status & Status.DELETE) == Status.DELETE);
			enumerable.ToList<AddPointModel>().ForEach(delegate(AddPointModel e)
			{
				this._addPoints.Remove(e);
			});
			AddPointModel apToDel = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == nameToDel && Status.NULL == (a.Status & Status.DELETE)).ElementAtOrDefault<AddPointModel>(0);
			if (apToDel.Status == Status.CREATE)
			{
				apToDel.Status = Status.CREATE | Status.DELETE;
			}
			else
			{
				apToDel.Status = Status.DELETE;
			}
			string targetName = string.Empty;
			switch (apToDel.Type)
			{
			case DefinitionType.FUNCTION:
				targetName = "FUNCTION";
				break;
			case DefinitionType.DIALOG:
				targetName = "DIALOG";
				break;
			case DefinitionType.REPORT:
				targetName = "REPORT";
				break;
			}
			TreeItem treeItem = this._nodes.ElementAtOrDefault<TreeItem>(0).Nodes.Where<TreeItem>((TreeItem t) => t.Type == DefinitionType.NULL && t.Name == targetName).ElementAtOrDefault<TreeItem>(0);
			if (treeItem == null)
			{
				throw new Exception(string.Format(Application.Current.FindResource("Message_StructureError") as string, apToDel.Name));
			}
			TreeItem treeItem2 = treeItem.Nodes.Where<TreeItem>((TreeItem n) => n.ReferenceName == apToDel.Name).ElementAtOrDefault<TreeItem>(0);
			treeItem.Remove(treeItem2);
		}

		// Token: 0x060001C3 RID: 451 RVA: 0x000087FC File Offset: 0x000069FC
		public void Add(AddPointModel funToAdd)
		{
			if (!funToAdd.IsSelfDefinition)
			{
				return;
			}
			IEnumerable<AddPointModel> enumerable = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == funToAdd.Name && (a.Status & Status.DELETE) == Status.NULL);
			if (enumerable.Count<AddPointModel>() > 0)
			{
				AddPointModel addPointModel = enumerable.ElementAt<AddPointModel>(0);
				this._addPoints.Remove(addPointModel);
			}
			funToAdd.IsLoaded = true;
			AddPointModel addPointModel2 = (from model in this._addPoints
				where model.SortIndex < int.MaxValue && model.Type != DefinitionType.NULL && (model.Status & Status.DELETE) == Status.NULL
				select model into m
				orderby m.SortIndex descending
				select m).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel2 != null)
			{
				funToAdd.SortIndex = addPointModel2.SortIndex + 1;
			}
			else
			{
				funToAdd.SortIndex = 1;
			}
			this._addPoints.Add(funToAdd);
			TreeItem treeItem = TreeNodeFactory.Parse(funToAdd);
			this._nodes.ElementAtOrDefault<TreeItem>(0).AddNode(treeItem);
			treeItem.PublishSelectedEvent();
		}

		// Token: 0x060001C4 RID: 452 RVA: 0x00008980 File Offset: 0x00006B80
		public void Modify(ModifyInfomation infoForModify)
		{
			IEnumerable<AddPointModel> enumerable = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == infoForModify.Source && (a.Status & Status.DELETE) == Status.DELETE);
			enumerable.ToList<AddPointModel>().ForEach(delegate(AddPointModel e)
			{
				this._addPoints.Remove(e);
			});
			AddPointModel addPointModel = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == infoForModify.Source && Status.NULL == (a.Status & Status.DELETE)).ElementAtOrDefault<AddPointModel>(0);
			if (!addPointModel.IsSelfDefinition)
			{
				return;
			}
			addPointModel.Status = Status.MODIFY;
			AddPointModel addPointModel2 = addPointModel.Clone();
			addPointModel2.Status = Status.DELETE;
			this._addPoints.Add(addPointModel2);
			addPointModel.FunctionName = infoForModify.Modified;
			this._nodes.ElementAtOrDefault<TreeItem>(0).Update(infoForModify.Source, TreeNodeFactory.Parse(addPointModel));
		}

		// Token: 0x060001C5 RID: 453 RVA: 0x00008AB0 File Offset: 0x00006CB0
		public void FunctionModify(ModifyFunctionInformation infoForModify)
		{
			IEnumerable<AddPointModel> enumerable = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.Name == infoForModify.Source && (a.Status & Status.DELETE) == Status.DELETE);
			enumerable.ToList<AddPointModel>().ForEach(delegate(AddPointModel e)
			{
				this._addPoints.Remove(e);
			});
			AddPointModel addPointModel = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.ID == infoForModify.ID && Status.NULL == (a.Status & Status.DELETE)).ElementAtOrDefault<AddPointModel>(0);
			addPointModel.Status = Status.DELETE;
			AddPointModel addPointModel2 = addPointModel.Clone();
			addPointModel2.Status = Status.MODIFY;
			addPointModel2.FunctionName = infoForModify.Modified;
			addPointModel2.Scope = infoForModify.Scope;
			addPointModel2.Description = infoForModify.Description;
			addPointModel2.Content = infoForModify.Content;
			if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsDiff && addPointModel.DiffBaseOnStandardModify == "Y")
			{
				addPointModel2.OriContent = addPointModel.OriContent;
			}
			this._addPoints.Add(addPointModel2);
			this._nodes.ElementAtOrDefault<TreeItem>(0).Update(addPointModel.Name, TreeNodeFactory.Parse(addPointModel2));
		}

		// Token: 0x060001C6 RID: 454 RVA: 0x00008BD8 File Offset: 0x00006DD8
		public void Clear()
		{
			for (int i = 0; i < this._addPoints.Count; i++)
			{
				AddPointModel addPointModel = this._addPoints[i];
				addPointModel.Dispose();
			}
			for (int j = 0; j < this._nodes.Count; j++)
			{
				TreeItem treeItem = this._nodes[j];
				treeItem.Dispose();
			}
			if (this._citeAddPoints != null)
			{
				this._citeAddPoints.Clear();
			}
			if (this._addPoints != null)
			{
				this._addPoints.Clear();
			}
			if (this._variables != null)
			{
				this._variables.Clear();
			}
			if (this._nodes != null)
			{
				this._nodes.Clear();
			}
			if (this._bgWorker != null)
			{
				if (this._bgWorker.IsBusy)
				{
					this._bgWorker.CancelAsync();
					if (this.parser != null)
					{
						this.parser.CancelJob();
					}
				}
				this._bgWorker.Dispose();
			}
			if (this._results != null)
			{
				this._results.Clear();
			}
			if (this._searchHistory != null)
			{
				this._searchHistory.Clear();
			}
		}

		// Token: 0x17000094 RID: 148
		// (get) Token: 0x060001C7 RID: 455 RVA: 0x00008CE7 File Offset: 0x00006EE7
		// (set) Token: 0x060001C8 RID: 456 RVA: 0x00008CEF File Offset: 0x00006EEF
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x17000095 RID: 149
		// (get) Token: 0x060001C9 RID: 457 RVA: 0x00008CF8 File Offset: 0x00006EF8
		// (set) Token: 0x060001CA RID: 458 RVA: 0x00008D50 File Offset: 0x00006F50
		public bool IsSectionModify
		{
			get
			{
				return this._tap != null && this._tap.Attribute("section_flag") != null && string.Equals(this._tap.Attribute("section_flag").Value, "Y", StringComparison.CurrentCultureIgnoreCase);
			}
			set
			{
				this._tap.SetAttributeValue("section_flag", value ? "Y" : "N");
				this.OnPropertyChanged("IsSectionModify");
			}
		}

		// Token: 0x17000096 RID: 150
		// (get) Token: 0x060001CB RID: 459 RVA: 0x00008D84 File Offset: 0x00006F84
		// (set) Token: 0x060001CC RID: 460 RVA: 0x00008DDC File Offset: 0x00006FDC
		public bool IsStdSectionVerify
		{
			get
			{
				return this._tap != null && this._tap.Attribute("std_section_verify") != null && string.Equals(this._tap.Attribute("std_section_verify").Value, "Y", StringComparison.CurrentCultureIgnoreCase);
			}
			set
			{
				this._tap.SetAttributeValue("std_section_verify", value ? "Y" : "N");
				this.OnPropertyChanged("std_section_verify");
			}
		}

		// Token: 0x17000097 RID: 151
		// (get) Token: 0x060001CD RID: 461 RVA: 0x00008E10 File Offset: 0x00007010
		public bool IsFreeStyle
		{
			get
			{
				return string.Equals(this._tap.Element("other").Element("free_style").Attribute("value")
					.Value, "Y", StringComparison.CurrentCultureIgnoreCase);
			}
		}

		// Token: 0x17000098 RID: 152
		// (get) Token: 0x060001CE RID: 462 RVA: 0x00008E65 File Offset: 0x00007065
		public string Type
		{
			get
			{
				return this._tap.Attribute("type").Value;
			}
		}

		// Token: 0x17000099 RID: 153
		// (get) Token: 0x060001CF RID: 463 RVA: 0x00008E81 File Offset: 0x00007081
		public bool IsStandard
		{
			get
			{
				return string.Equals(this._tap.Attribute("std_prog").Value, this._tap.Attribute("prog").Value);
			}
		}

		// Token: 0x1700009A RID: 154
		// (get) Token: 0x060001D0 RID: 464 RVA: 0x00008EC1 File Offset: 0x000070C1
		public string ModuleName
		{
			get
			{
				if (this._tap.Attribute("module") != null)
				{
					return this._tap.Attribute("module").Value;
				}
				return null;
			}
		}

		// Token: 0x1700009B RID: 155
		// (get) Token: 0x060001D1 RID: 465 RVA: 0x00008EF6 File Offset: 0x000070F6
		public string Ver
		{
			get
			{
				if (this._tap.Attribute("ver") != null)
				{
					return this._tap.Attribute("ver").Value;
				}
				return null;
			}
		}

		// Token: 0x1700009C RID: 156
		// (get) Token: 0x060001D2 RID: 466 RVA: 0x00008F2B File Offset: 0x0000712B
		public string ENV
		{
			get
			{
				if (this._tap.Attribute("env") != null)
				{
					return this._tap.Attribute("env").Value;
				}
				return string.Empty;
			}
		}

		// Token: 0x1700009D RID: 157
		// (get) Token: 0x060001D3 RID: 467 RVA: 0x00008F64 File Offset: 0x00007164
		public bool IsDiff
		{
			get
			{
				return SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff;
			}
		}

		// Token: 0x1700009E RID: 158
		// (get) Token: 0x060001D4 RID: 468 RVA: 0x00008F7B File Offset: 0x0000717B
		public string StdToCus
		{
			get
			{
				if (this._tap.Attribute("std_to_cus") != null)
				{
					return this._tap.Attribute("std_to_cus").Value;
				}
				return string.Empty;
			}
		}

		// Token: 0x1700009F RID: 159
		// (get) Token: 0x060001D5 RID: 469 RVA: 0x00008FB4 File Offset: 0x000071B4
		public string Topind
		{
			get
			{
				if (this._tap.Attribute("topind") != null)
				{
					return this._tap.Attribute("topind").Value;
				}
				return string.Empty;
			}
		}

		// Token: 0x060001D6 RID: 470 RVA: 0x00008FF0 File Offset: 0x000071F0
		public string GetCodeTemplate()
		{
			XElement xelement = this._tap.Element("other");
			if (xelement == null)
			{
				return null;
			}
			return xelement.Element("code_template").Attribute("value").Value;
		}

		// Token: 0x060001D7 RID: 471 RVA: 0x0000903C File Offset: 0x0000723C
		public void SetCodeTemplate(string template)
		{
			try
			{
				this._tap.Element("other").Element("code_template").Attribute("value")
					.Value = template;
				this._tap.Element("other").Element("code_template").Attribute("status")
					.Value = "u";
			}
			catch
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_SetProgTypeError") as string, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK);
				return;
			}
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x170000A0 RID: 160
		// (get) Token: 0x060001D8 RID: 472 RVA: 0x00009120 File Offset: 0x00007320
		// (set) Token: 0x060001D9 RID: 473 RVA: 0x0000916C File Offset: 0x0000736C
		public string StartArg
		{
			get
			{
				XElement xelement = this._tap.Element("other");
				if (xelement == null)
				{
					return null;
				}
				return xelement.Element("start_arg").Attribute("value").Value;
			}
			set
			{
				this._tap.Element("other").Element("start_arg").Attribute("value")
					.Value = value;
				this._tap.Element("other").Element("start_arg").Attribute("status")
					.Value = "u";
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
			}
		}

		// Token: 0x060001DA RID: 474 RVA: 0x0000920C File Offset: 0x0000740C
		public string GetCodeStartArg()
		{
			XElement xelement = this._tap.Element("other");
			if (xelement == null)
			{
				return null;
			}
			if (xelement.Element("start_arg") == null)
			{
				return null;
			}
			return xelement.Element("start_arg").Attribute("value").Value;
		}

		// Token: 0x060001DB RID: 475 RVA: 0x0000926C File Offset: 0x0000746C
		public void SetCodeStartArg(string arg)
		{
			try
			{
				this._tap.Element("other").Element("start_arg").Attribute("value")
					.Value = arg;
				this._tap.Element("other").Element("start_arg").Attribute("status")
					.Value = "u";
			}
			catch
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_SetArgError") as string, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK);
				return;
			}
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x060001DC RID: 476 RVA: 0x00009350 File Offset: 0x00007550
		public void SetFreeStyle()
		{
			try
			{
				this._tap.Element("other").Element("free_style").Attribute("value")
					.Value = "Y";
				this._tap.Element("other").Element("free_style").Attribute("status")
					.Value = "u";
			}
			catch
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_SetFreeStyleError") as string, "Error", MessageBoxButton.OK);
				return;
			}
			this.OnPropertyChanged("IsFreeStyle");
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x170000A1 RID: 161
		// (get) Token: 0x060001DD RID: 477 RVA: 0x00009434 File Offset: 0x00007634
		public string login_user
		{
			get
			{
				if (this._tap.Attribute("login_user") != null)
				{
					return this._tap.Attribute("login_user").Value;
				}
				return null;
			}
		}

		// Token: 0x170000A2 RID: 162
		// (get) Token: 0x060001DE RID: 478 RVA: 0x00009469 File Offset: 0x00007669
		public ReadOnlyCollection<AddPointModel> DiffAddPoints
		{
			get
			{
				return new ReadOnlyCollection<AddPointModel>(this._diffAddPoints);
			}
		}

		// Token: 0x170000A3 RID: 163
		// (set) Token: 0x060001DF RID: 479 RVA: 0x00009478 File Offset: 0x00007678
		public string DiffAddPoint
		{
			set
			{
				if (this._diffAddPoints == null)
				{
					this._diffAddPoints = new List<AddPointModel>();
				}
				this._diffAddPoints.Clear();
				if (this._difftap != null)
				{
					this._difftap.RemoveAll();
				}
				this._difftap = XElement.Parse(value);
				foreach (XElement xelement in this._difftap.Elements("point"))
				{
					AddPointModel addPointModel = AddPointModel.CreateDiffAddpoint(xelement, this.ProgramKey);
					if (addPointModel != null)
					{
						this._diffAddPoints.Add(addPointModel);
					}
				}
			}
		}

		// Token: 0x170000A4 RID: 164
		// (get) Token: 0x060001E0 RID: 480 RVA: 0x00009528 File Offset: 0x00007728
		public ReadOnlyCollection<AddPointModel> AddPoints
		{
			get
			{
				return new ReadOnlyCollection<AddPointModel>(this._addPoints);
			}
		}

		// Token: 0x170000A5 RID: 165
		// (get) Token: 0x060001E1 RID: 481 RVA: 0x00009535 File Offset: 0x00007735
		public ReadOnlyCollection<AddPointModel> CiteAddPoints
		{
			get
			{
				return new ReadOnlyCollection<AddPointModel>(this._citeAddPoints ?? new List<AddPointModel>());
			}
		}

		// Token: 0x170000A6 RID: 166
		// (get) Token: 0x060001E2 RID: 482 RVA: 0x0000954B File Offset: 0x0000774B
		public XElement TAP
		{
			get
			{
				return this._tap;
			}
		}

		// Token: 0x170000A7 RID: 167
		// (get) Token: 0x060001E3 RID: 483 RVA: 0x00009553 File Offset: 0x00007753
		// (set) Token: 0x060001E4 RID: 484 RVA: 0x0000955B File Offset: 0x0000775B
		public string UpdatedAddPoint { get; set; }

		// Token: 0x170000A8 RID: 168
		// (get) Token: 0x060001E5 RID: 485 RVA: 0x00009580 File Offset: 0x00007780
		// (set) Token: 0x060001E6 RID: 486 RVA: 0x000097F8 File Offset: 0x000079F8
		public string AddPoint
		{
			get
			{
				XElement xelement = new XElement(this._tap.Element("other"));
				this._tap.RemoveNodes();
				XElement xelement2 = new XElement(this._tap);
				XElement xelement3 = XElement.Parse(xelement.ToString());
				int i = 0;
				while (i < xelement3.Elements().Count<XElement>())
				{
					XElement xelement4 = xelement3.Elements().ElementAt<XElement>(i);
					if (string.IsNullOrEmpty(xelement4.Attribute("status").Value))
					{
						xelement4.Remove();
					}
					else
					{
						i++;
					}
				}
				if (xelement3.Elements().Count<XElement>() > 0)
				{
					xelement2.Add(xelement3);
				}
				this._tap.Add(xelement);
				IEnumerable<AddPointModel> enumerable = this.AddPoints.Where<AddPointModel>((AddPointModel model) => (model.Status & Status.DELETE) == Status.DELETE);
				foreach (AddPointModel addPointModel in enumerable)
				{
					XElement xelement5 = addPointModel.ToXML();
					if (xelement5 != null)
					{
						this._tap.Add(xelement5);
						xelement2.Add(xelement5);
					}
				}
				IEnumerable<AddPointModel> enumerable2 = this.AddPoints.Where<AddPointModel>((AddPointModel model) => (model.Status & Status.DELETE) == Status.NULL);
				foreach (AddPointModel addPointModel2 in enumerable2)
				{
					XElement xelement6 = addPointModel2.ToXML();
					if (xelement6 != null)
					{
						this._tap.Add(xelement6);
						if ((addPointModel2.Status & Status.MODIFY) == Status.MODIFY)
						{
							xelement2.Add(xelement6);
						}
					}
				}
				foreach (SectionModel sectionModel in this.Sections)
				{
					XElement xelement7 = sectionModel.ToXElement();
					if (xelement7 != null)
					{
						this._tap.Add(xelement7);
						if ("u" == sectionModel.Status)
						{
							xelement2.Add(xelement7);
						}
					}
				}
				XDocument xdocument = new XDocument();
				xdocument.Add(this._tap);
				XDocument xdocument2 = new XDocument();
				xdocument2.Add(xelement2);
				this.UpdatedAddPoint = xdocument2.ToString();
				return xdocument.ToString();
			}
			set
			{
				if (this._addPoints == null)
				{
					this._addPoints = new List<AddPointModel>();
				}
				this._addPoints.Clear();
				if (this._tap != null)
				{
					this._tap.RemoveAll();
				}
				this._tap = XElement.Parse(value);
				foreach (XElement xelement in this._tap.Elements("point"))
				{
					AddPointModel addPointModel = AddPointModel.Create(xelement, this.ProgramKey);
					if (addPointModel != null)
					{
						this._addPoints.Add(addPointModel);
					}
				}
				foreach (XElement xelement2 in this._tap.Elements("section"))
				{
					SectionModel sectionModel = new SectionModel(this.ProgramKey, xelement2);
					this.Sections.Add(sectionModel);
				}
			}
		}

		// Token: 0x170000A9 RID: 169
		// (get) Token: 0x060001E7 RID: 487 RVA: 0x00009910 File Offset: 0x00007B10
		// (set) Token: 0x060001E8 RID: 488 RVA: 0x00009918 File Offset: 0x00007B18
		public string FullCode { get; set; }

		// Token: 0x060001E9 RID: 489 RVA: 0x0000993C File Offset: 0x00007B3C
		public AddPointModel Initial(string name)
		{
			AddPointModel addPointModel = this._addPoints.Where<AddPointModel>((AddPointModel m) => m.Name.Equals(name)).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel == null)
			{
				addPointModel = new AddPointModel(this.ProgramKey);
				addPointModel.Name = name;
				addPointModel.IsLoaded = true;
				this._addPoints.Add(addPointModel);
			}
			return addPointModel;
		}

		// Token: 0x170000AA RID: 170
		// (get) Token: 0x060001EA RID: 490 RVA: 0x000099A3 File Offset: 0x00007BA3
		public ReadOnlyCollection<Variable> Variables
		{
			get
			{
				return new ReadOnlyCollection<Variable>(this._variables);
			}
		}

		// Token: 0x060001EB RID: 491 RVA: 0x000099B0 File Offset: 0x00007BB0
		public void Add(Variable variable)
		{
			if (this._variables.Contains(variable))
			{
				this._variables.Remove(variable);
			}
			this._variables.Add(variable);
		}

		// Token: 0x170000AB RID: 171
		// (get) Token: 0x060001EC RID: 492 RVA: 0x000099D9 File Offset: 0x00007BD9
		public ReadOnlyObservableCollection<TreeItem> Nodes
		{
			get
			{
				return new ReadOnlyObservableCollection<TreeItem>(this._nodes);
			}
		}

		// Token: 0x170000AC RID: 172
		// (get) Token: 0x060001ED RID: 493 RVA: 0x000099E6 File Offset: 0x00007BE6
		// (set) Token: 0x060001EE RID: 494 RVA: 0x000099F0 File Offset: 0x00007BF0
		public string TGL
		{
			get
			{
				return this._tgl;
			}
			set
			{
				if (string.IsNullOrEmpty(value))
				{
					throw new Exception(Application.Current.FindResource("CE_TGLIsEmpty") as string);
				}
				bool flag = string.IsNullOrEmpty(this._tgl);
				this._tgl = value;
				if (flag)
				{
					this.CreateCodeStructure();
				}
			}
		}

		// Token: 0x060001EF RID: 495 RVA: 0x00009CE8 File Offset: 0x00007EE8
		private IEnumerable<TreeItem> Find(TreeItem root, string title)
		{
			foreach (TreeItem item in root.Nodes)
			{
				if (item.Name.Equals(title, StringComparison.InvariantCultureIgnoreCase))
				{
					yield return item;
				}
				if (item.HasChild)
				{
					foreach (TreeItem sub in this.Find(item, title))
					{
						yield return sub;
					}
				}
			}
			yield break;
		}

		// Token: 0x060001F0 RID: 496 RVA: 0x00009F4C File Offset: 0x0000814C
		public IEnumerable<TreeItem> Find(string title)
		{
			foreach (TreeItem item in this.Nodes)
			{
				foreach (TreeItem i in this.Find(item, title))
				{
					yield return i;
				}
			}
			yield break;
		}

		// Token: 0x170000AD RID: 173
		// (get) Token: 0x060001F1 RID: 497 RVA: 0x00009F70 File Offset: 0x00008170
		// (set) Token: 0x060001F2 RID: 498 RVA: 0x00009F78 File Offset: 0x00008178
		public string FullContext
		{
			get
			{
				return this._fullContext;
			}
			set
			{
				this._fullContext = value;
			}
		}

		// Token: 0x170000AE RID: 174
		// (get) Token: 0x060001F3 RID: 499 RVA: 0x00009F84 File Offset: 0x00008184
		public bool AllowChangeToTopstd
		{
			get
			{
				return this.TAP.Attribute("login_user") != null && (this.TAP.Attribute("login_user").Value == "topapp" || this.TAP.Attribute("login_user").Value == "topman") && this.TAP.Attribute("std_to_cus") != null && this.TAP.Attribute("std_to_cus").Value == "Y";
			}
		}

		// Token: 0x170000AF RID: 175
		// (get) Token: 0x060001F4 RID: 500 RVA: 0x0000A034 File Offset: 0x00008234
		// (set) Token: 0x060001F5 RID: 501 RVA: 0x0000A03C File Offset: 0x0000823C
		public bool IsTopstdMode { get; set; }

		// Token: 0x060001F6 RID: 502 RVA: 0x0000A074 File Offset: 0x00008274
		public bool IsADPEditable(Guid guid)
		{
			AddPointModel addPointModel = this.AddPoints.Where<AddPointModel>((AddPointModel a) => a.ID == guid && (a.Status & Status.DELETE) != Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
			return addPointModel != null && addPointModel.IsEditable;
		}

		// Token: 0x060001F7 RID: 503 RVA: 0x0000A0D4 File Offset: 0x000082D4
		public bool IsSectionEditable(Guid guid)
		{
			SectionModel sectionModel = this.Sections.Where<SectionModel>((SectionModel s) => s.ID == guid).ElementAtOrDefault<SectionModel>(0);
			return sectionModel != null && sectionModel.IsEditable;
		}

		// Token: 0x060001F8 RID: 504 RVA: 0x0000A170 File Offset: 0x00008370
		private void CreateCodeStructure()
		{
			this._nodes.Clear();
			TreeItem rootNode = new TreeItem(string.Format("{0}/{1}", this.ModuleName, this.ProgramKey.Program));
			rootNode.ProgramKey = this.ProgramKey;
			this._nodes.Add(rootNode);
			this._functionTypes.Clear();
			Dictionary<int, string> dictionary = new Dictionary<int, string>();
			Regex regex = new Regex("{<point\\s+name=\"(other.function)\".*\\s*/>}", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			if (regex.IsMatch(this._tgl))
			{
				Match match = regex.Match(this._tgl);
				dictionary.Add(match.Index, "FUNCTION");
			}
			Regex regex2 = new Regex("{<point\\s+name=\"(other.dialog)\".*\\s*/>}", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			if (regex2.IsMatch(this._tgl))
			{
				Match match2 = regex2.Match(this._tgl);
				dictionary.Add(match2.Index, "DIALOG");
			}
			Regex regex3 = new Regex("{<point\\s+name=\"(other.report)\".*\\s*/>}", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			if (regex3.IsMatch(this._tgl))
			{
				Match match3 = regex3.Match(this._tgl);
				dictionary.Add(match3.Index, "REPORT");
			}
			IOrderedEnumerable<KeyValuePair<int, string>> orderedEnumerable = dictionary.OrderBy<KeyValuePair<int, string>, int>((KeyValuePair<int, string> item) => item.Key);
			for (int i = 0; i < dictionary.Count; i++)
			{
				this._functionTypes.Add(orderedEnumerable.ElementAtOrDefault<KeyValuePair<int, string>>(i).Value);
			}
			try
			{
				List<TreeItem> list = TreeNodeFactory.Parse(this._tgl, this.ProgramKey);
				list.ForEach(delegate(TreeItem res)
				{
					rootNode.AddNode(res);
				});
			}
			catch
			{
				throw new FormatException(Application.Current.FindResource("Message_ParseError") as string);
			}
			this._functionTypes.ForEach(delegate(string type)
			{
				rootNode.AddNode(new TreeItem(type)
				{
					IsFolder = true,
					ProgramKey = this.ProgramKey
				});
			});
		}

		// Token: 0x170000B0 RID: 176
		// (get) Token: 0x060001F9 RID: 505 RVA: 0x0000A370 File Offset: 0x00008570
		public DiffModel DiffModel
		{
			get
			{
				if (this._diffModel == null && SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
				{
					this._diffModel = new DiffModel(XElement.Parse(SettingManager.Get().GetTzpManger(this.ProgramKey).ADT), XElement.Parse(SettingManager.Get().GetTzpManger(this.ProgramKey).TAP), this.ProgramKey);
				}
				return this._diffModel;
			}
		}

		// Token: 0x170000B1 RID: 177
		// (get) Token: 0x060001FA RID: 506 RVA: 0x0000A3E7 File Offset: 0x000085E7
		// (set) Token: 0x060001FB RID: 507 RVA: 0x0000A3EF File Offset: 0x000085EF
		public List<DiffColor> DiffColorArea
		{
			get
			{
				return this._diffColorArea;
			}
			set
			{
				this._diffColorArea = value;
			}
		}

		// Token: 0x170000B2 RID: 178
		// (get) Token: 0x060001FC RID: 508 RVA: 0x0000A3F8 File Offset: 0x000085F8
		// (set) Token: 0x060001FD RID: 509 RVA: 0x0000A400 File Offset: 0x00008600
		public int DiffTotalLineLumber
		{
			get
			{
				return this._diffTotalLineNumber;
			}
			set
			{
				this._diffTotalLineNumber = value;
			}
		}

		// Token: 0x170000B3 RID: 179
		// (get) Token: 0x060001FE RID: 510 RVA: 0x0000A409 File Offset: 0x00008609
		// (set) Token: 0x060001FF RID: 511 RVA: 0x0000A411 File Offset: 0x00008611
		public List<DiffColor> BaseOnStandardDiffColorArea
		{
			get
			{
				return this._baseOnStandardDiffColorArea;
			}
			set
			{
				this._baseOnStandardDiffColorArea = value;
			}
		}

		// Token: 0x06000200 RID: 512 RVA: 0x0000A458 File Offset: 0x00008658
		public string FindDiffSourceContetByName(string name)
		{
			XElement diff_TAP = SettingManager.Get().GetTzpManger(this.ProgramKey).DIFF_TAP;
			if (diff_TAP != null)
			{
				XElement xelement = (from p in diff_TAP.Elements("point")
					where p.Attribute("name") != null && p.Attribute("name").Value == name
					select p).FirstOrDefault<XElement>();
				if (xelement != null)
				{
					return xelement.Value;
				}
			}
			return null;
		}

		// Token: 0x06000201 RID: 513 RVA: 0x0000A504 File Offset: 0x00008704
		public XElement FindDiffSourceByName(string name)
		{
			XElement diff_TAP = SettingManager.Get().GetTzpManger(this.ProgramKey).DIFF_TAP;
			if (diff_TAP != null)
			{
				XElement xelement = (from p in diff_TAP.Elements("point")
					where p.Attribute("name") != null && p.Attribute("name").Value == name
					select p).FirstOrDefault<XElement>();
				if (xelement != null)
				{
					return xelement;
				}
			}
			return null;
		}

		// Token: 0x170000B4 RID: 180
		// (get) Token: 0x06000202 RID: 514 RVA: 0x0000A56B File Offset: 0x0000876B
		// (set) Token: 0x06000203 RID: 515 RVA: 0x0000A573 File Offset: 0x00008773
		public bool VerifyAdjustFunctionSort { get; set; }

		// Token: 0x06000204 RID: 516 RVA: 0x0000A5BC File Offset: 0x000087BC
		public void RefreshTreeNodes()
		{
			if (string.IsNullOrEmpty(this.FullContext))
			{
				return;
			}
			if (this._bgWorker.IsBusy)
			{
				DesignerMessageBox.Show(Application.Current.FindResource("Message_ParsingUnFinish") as string, Application.Current.FindResource("Message_Error") as string, MessageBoxButton.OK, MessageBoxImage.Asterisk);
				return;
			}
			TreeItem rootNode = this._nodes.ElementAt<TreeItem>(0);
			rootNode.Clear();
			if (!rootNode.HasChild)
			{
				this._functionTypes.ForEach(delegate(string type)
				{
					rootNode.AddNode(new TreeItem(type)
					{
						IsFolder = true,
						ProgramKey = this.ProgramKey
					});
				});
			}
			this._progrocess = 0;
			this._bgWorker.WorkerReportsProgress = true;
			this._bgWorker.WorkerSupportsCancellation = true;
			this._bgWorker.ProgressChanged += this.worker_ProgressChanged;
			this._bgWorker.RunWorkerCompleted += this.worker_RunWorkerCompleted;
			this._bgWorker.DoWork += this.worker_DoWork;
			this._bgWorker.RunWorkerAsync();
		}

		// Token: 0x06000205 RID: 517 RVA: 0x0000A6DC File Offset: 0x000088DC
		private void worker_RunWorkerCompleted(object sender, RunWorkerCompletedEventArgs e)
		{
			this._bgWorker.ProgressChanged -= this.worker_ProgressChanged;
			this._bgWorker.RunWorkerCompleted -= this.worker_RunWorkerCompleted;
			this._bgWorker.DoWork -= this.worker_DoWork;
			this._bgWorker.Dispose();
		}

		// Token: 0x06000206 RID: 518 RVA: 0x0000A780 File Offset: 0x00008980
		private void worker_ProgressChanged(object sender, ProgressChangedEventArgs e)
		{
			TreeItem item = e.UserState as TreeItem;
			if (item == null)
			{
				return;
			}
			AddPointModel addPointModel = this._addPoints.Where<AddPointModel>((AddPointModel m) => m.FunctionNameWithoutParameter == item.Name && m.Type == item.Type && (m.Status & Status.DELETE) != Status.DELETE).ElementAtOrDefault<AddPointModel>(0);
			if (addPointModel != null)
			{
				TreeNodeFactory.CloneFrom(ref item, addPointModel);
				this.Nodes[0].AddNode(item);
			}
		}

		// Token: 0x06000207 RID: 519 RVA: 0x0000A7F4 File Offset: 0x000089F4
		private void worker_DoWork(object sender, DoWorkEventArgs e)
		{
			try
			{
				List<TreeItem> list = TreeNodeFactory.Parse(this.FullContext, this.ProgramKey);
				foreach (TreeItem treeItem in list)
				{
					this._bgWorker.ReportProgress(this._progrocess++, treeItem);
				}
			}
			catch
			{
				e.Cancel = true;
				this._bgWorker.ReportProgress(100);
			}
		}

		// Token: 0x170000B5 RID: 181
		// (get) Token: 0x06000208 RID: 520 RVA: 0x0000A890 File Offset: 0x00008A90
		// (set) Token: 0x06000209 RID: 521 RVA: 0x0000A898 File Offset: 0x00008A98
		public string MajorAbnormalSearchKey { get; set; }

		// Token: 0x170000B6 RID: 182
		// (get) Token: 0x0600020A RID: 522 RVA: 0x0000A8A9 File Offset: 0x00008AA9
		public List<NormalizationSearchResult> Results
		{
			get
			{
				return this._results.OrderBy<NormalizationSearchResult, int>((NormalizationSearchResult f) => f.StartOffset).ToList<NormalizationSearchResult>();
			}
		}

		// Token: 0x170000B7 RID: 183
		// (get) Token: 0x0600020B RID: 523 RVA: 0x0000A8D8 File Offset: 0x00008AD8
		// (set) Token: 0x0600020C RID: 524 RVA: 0x0000A8E0 File Offset: 0x00008AE0
		public NormalizationSearchResult CurrentSearchResult
		{
			get
			{
				return this._currentSearchResult;
			}
			set
			{
				this._currentSearchResult = value;
			}
		}

		// Token: 0x170000B8 RID: 184
		// (get) Token: 0x0600020D RID: 525 RVA: 0x0000A8E9 File Offset: 0x00008AE9
		// (set) Token: 0x0600020E RID: 526 RVA: 0x0000A8F1 File Offset: 0x00008AF1
		public ObservableCollection<string> SearchHistory
		{
			get
			{
				return this._searchHistory;
			}
			set
			{
				this._searchHistory = value;
			}
		}

		// Token: 0x0600020F RID: 527 RVA: 0x0000A8FA File Offset: 0x00008AFA
		public void ClearNormalizationSearchResult()
		{
			this._results.Clear();
		}

		// Token: 0x06000210 RID: 528 RVA: 0x0000A907 File Offset: 0x00008B07
		public void AddNormalizationSearchResult(NormalizationSearchResult result)
		{
			this._results.Add(result);
		}

		// Token: 0x1400000C RID: 12
		// (add) Token: 0x06000211 RID: 529 RVA: 0x0000A918 File Offset: 0x00008B18
		// (remove) Token: 0x06000212 RID: 530 RVA: 0x0000A950 File Offset: 0x00008B50
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000213 RID: 531 RVA: 0x0000A985 File Offset: 0x00008B85
		private void OnPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x040000B3 RID: 179
		private List<string> _functionTypes = new List<string>();

		// Token: 0x040000B4 RID: 180
		private List<AddPointModel> _diffAddPoints = new List<AddPointModel>();

		// Token: 0x040000B5 RID: 181
		private XElement _difftap;

		// Token: 0x040000B6 RID: 182
		private List<AddPointModel> _addPoints = new List<AddPointModel>();

		// Token: 0x040000B7 RID: 183
		private List<AddPointModel> _citeAddPoints;

		// Token: 0x040000B8 RID: 184
		public List<SectionModel> Sections = new List<SectionModel>();

		// Token: 0x040000B9 RID: 185
		private XElement _tap;

		// Token: 0x040000BA RID: 186
		private List<Variable> _variables = new List<Variable>();

		// Token: 0x040000BB RID: 187
		private TreeItemCollection _nodes = new TreeItemCollection();

		// Token: 0x040000BC RID: 188
		private string _tgl = string.Empty;

		// Token: 0x040000BD RID: 189
		private string _fullContext = string.Empty;

		// Token: 0x040000BE RID: 190
		private DiffModel _diffModel;

		// Token: 0x040000BF RID: 191
		private List<DiffColor> _diffColorArea = new List<DiffColor>();

		// Token: 0x040000C0 RID: 192
		private int _diffTotalLineNumber;

		// Token: 0x040000C1 RID: 193
		private List<DiffColor> _baseOnStandardDiffColorArea = new List<DiffColor>();

		// Token: 0x040000C2 RID: 194
		private BackgroundWorker _bgWorker;

		// Token: 0x040000C3 RID: 195
		private int _progrocess;

		// Token: 0x040000C4 RID: 196
		private FglParser parser;

		// Token: 0x040000C5 RID: 197
		public List<NormalizationSearchResult> _results = new List<NormalizationSearchResult>();

		// Token: 0x040000C6 RID: 198
		private NormalizationSearchResult _currentSearchResult;

		// Token: 0x040000C7 RID: 199
		private ObservableCollection<string> _searchHistory = new ObservableCollection<string>();
	}
}
