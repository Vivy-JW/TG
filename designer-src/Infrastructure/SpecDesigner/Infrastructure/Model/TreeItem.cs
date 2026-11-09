using System;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Input;
using SpecDesigner.Infrastructure.Event;
using SpecDesigner.Infrastructure.Extension;
using SpecDesigner.Infrastructure.Helper;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000042 RID: 66
	public class TreeItem : IFunctionModel, INotifyPropertyChanged, IDisposable, IDragable, IDropable
	{
		// Token: 0x06000181 RID: 385 RVA: 0x00007698 File Offset: 0x00005898
		public TreeItem(string name)
		{
			this.Scope = Scope.NULL;
			this.Name = name;
		}

		// Token: 0x1700007D RID: 125
		// (get) Token: 0x06000182 RID: 386 RVA: 0x000076F2 File Offset: 0x000058F2
		// (set) Token: 0x06000183 RID: 387 RVA: 0x0000771C File Offset: 0x0000591C
		public int SortIndex
		{
			get
			{
				return this._sorIndex;
			}
			set
			{
				if (this._sorIndex == value)
				{
					return;
				}
				this._sorIndex = value;
				AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.Name == this.ReferenceName && (m.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
				if (addPointModel != null)
				{
					addPointModel.SortIndex = this._sorIndex;
				}
			}
		}

		// Token: 0x1700007E RID: 126
		// (get) Token: 0x06000184 RID: 388 RVA: 0x00007776 File Offset: 0x00005976
		// (set) Token: 0x06000185 RID: 389 RVA: 0x0000777E File Offset: 0x0000597E
		public string ReferenceName { get; set; }

		// Token: 0x1700007F RID: 127
		// (get) Token: 0x06000186 RID: 390 RVA: 0x00007787 File Offset: 0x00005987
		// (set) Token: 0x06000187 RID: 391 RVA: 0x00007790 File Offset: 0x00005990
		public string Name
		{
			get
			{
				return this._name;
			}
			set
			{
				RegexOptions regexOptions = RegexOptions.IgnoreCase | RegexOptions.Multiline;
				Regex regex = new Regex("^(?<scope>PRIVATE\\s+|PUBLIC\\s+|)(?<type>FUNCTION|DIALOG|REPORT)\\s+(?<name>.*)(?:\\([^\\)]*\\))", regexOptions);
				new Regex("\\s+", regexOptions);
				if (regex.IsMatch(value))
				{
					Match match = regex.Match(value);
					if (string.IsNullOrEmpty(this._name))
					{
						string text;
						if ((text = match.Groups["type"].Value.ToUpper()) != null)
						{
							if (text == "FUNCTION")
							{
								this.Type = DefinitionType.FUNCTION;
								goto IL_00AA;
							}
							if (text == "DIALOG")
							{
								this.Type = DefinitionType.DIALOG;
								goto IL_00AA;
							}
							if (text == "REPORT")
							{
								this.Type = DefinitionType.REPORT;
								goto IL_00AA;
							}
						}
						this.Type = DefinitionType.NULL;
					}
					IL_00AA:
					string text2;
					if ((text2 = match.Groups["scope"].Value.TrimEnd(new char[0]).ToUpper()) != null)
					{
						if (text2 == "PUBLIC")
						{
							this.Scope = Scope.PUBLIC;
							goto IL_010B;
						}
						if (text2 == "PRIVATE")
						{
							this.Scope = Scope.PRIVATE;
							goto IL_010B;
						}
					}
					this.Scope = Scope.NULL;
					IL_010B:
					this.Description = value.Substring(0, match.Groups[0].Index).TrimEnd(new char[0]);
					this._name = match.Groups["name"].Value;
				}
				else
				{
					this._name = value;
				}
				this.RaisePropertyChanged("Name");
				this.RaisePropertyChanged("DisplayName");
			}
		}

		// Token: 0x17000080 RID: 128
		// (get) Token: 0x06000188 RID: 392 RVA: 0x0000790C File Offset: 0x00005B0C
		public string DisplayName
		{
			get
			{
				if (string.IsNullOrEmpty(this._displayName))
				{
					if (this.Name.Contains(" FIELD "))
					{
						string[] array = this.Name.Split(new char[] { ' ' });
						this._displayName = array[0];
						for (int i = 1; i < array.Length; i++)
						{
							string text = array[i];
							this._displayName = string.Format("{0,-6} {1,-5}", this._displayName, text);
						}
					}
					else
					{
						this._displayName = this.Name;
					}
				}
				return this._displayName;
			}
		}

		// Token: 0x06000189 RID: 393 RVA: 0x00007998 File Offset: 0x00005B98
		private void GetScope(ref string sb)
		{
			Regex regex = new Regex("(?<type>^public|^private)", RegexOptions.IgnoreCase | RegexOptions.Multiline);
			if (regex.IsMatch(sb))
			{
				MatchCollection matchCollection = regex.Matches(sb);
				string text;
				if ((text = matchCollection[0].Groups["type"].Value.ToUpper()) != null && text == "PRIVATE")
				{
					this.Scope = Scope.PRIVATE;
				}
				else
				{
					this.Scope = Scope.PUBLIC;
				}
				sb = regex.Replace(sb, "", 1);
			}
		}

		// Token: 0x17000081 RID: 129
		// (get) Token: 0x0600018A RID: 394 RVA: 0x00007A18 File Offset: 0x00005C18
		public string SearchName
		{
			get
			{
				if (this.Type == DefinitionType.NULL || this.Name == "MAIN")
				{
					return this.Name;
				}
				return string.Format("{0} {1}(", this.Type.Description(), this.Name);
			}
		}

		// Token: 0x17000082 RID: 130
		// (get) Token: 0x0600018B RID: 395 RVA: 0x00007A66 File Offset: 0x00005C66
		// (set) Token: 0x0600018C RID: 396 RVA: 0x00007A70 File Offset: 0x00005C70
		public TreeItem Parent
		{
			get
			{
				return this._parent;
			}
			set
			{
				this._parent = value;
				this.Level = this.Parent.Level + 1;
				foreach (TreeItem treeItem in this._nodes)
				{
					treeItem.Level = this.Level + 1;
				}
				if (this.Level > 1)
				{
					this.IsExpanded = false;
				}
				else
				{
					this.IsExpanded = true;
				}
				this.RaisePropertyChanged("Parent");
			}
		}

		// Token: 0x17000083 RID: 131
		// (get) Token: 0x0600018D RID: 397 RVA: 0x00007B04 File Offset: 0x00005D04
		// (set) Token: 0x0600018E RID: 398 RVA: 0x00007B29 File Offset: 0x00005D29
		public string Description
		{
			get
			{
				if (this.Type == DefinitionType.NULL)
				{
					return null;
				}
				if (string.IsNullOrEmpty(this._description))
				{
					return "EMPTY";
				}
				return this._description;
			}
			set
			{
				this._description = value;
				this.RaisePropertyChanged("Description");
			}
		}

		// Token: 0x17000084 RID: 132
		// (get) Token: 0x0600018F RID: 399 RVA: 0x00007B3D File Offset: 0x00005D3D
		// (set) Token: 0x06000190 RID: 400 RVA: 0x00007B45 File Offset: 0x00005D45
		public Scope Scope
		{
			get
			{
				return this._scope;
			}
			set
			{
				this._scope = value;
				this.RaisePropertyChanged("Scope");
			}
		}

		// Token: 0x17000085 RID: 133
		// (get) Token: 0x06000191 RID: 401 RVA: 0x00007B59 File Offset: 0x00005D59
		// (set) Token: 0x06000192 RID: 402 RVA: 0x00007B61 File Offset: 0x00005D61
		public bool IsVisible
		{
			get
			{
				return this._isVisible;
			}
			set
			{
				if (this._isVisible == value)
				{
					return;
				}
				this._isVisible = value;
				this.RaisePropertyChanged("IsVisible");
			}
		}

		// Token: 0x17000086 RID: 134
		// (get) Token: 0x06000193 RID: 403 RVA: 0x00007B7F File Offset: 0x00005D7F
		// (set) Token: 0x06000194 RID: 404 RVA: 0x00007B87 File Offset: 0x00005D87
		public DefinitionType Type { get; set; }

		// Token: 0x17000087 RID: 135
		// (get) Token: 0x06000195 RID: 405 RVA: 0x00007B90 File Offset: 0x00005D90
		// (set) Token: 0x06000196 RID: 406 RVA: 0x00007B98 File Offset: 0x00005D98
		public bool IsFolder { get; set; }

		// Token: 0x17000088 RID: 136
		// (get) Token: 0x06000197 RID: 407 RVA: 0x00007BA1 File Offset: 0x00005DA1
		public ReadOnlyObservableCollection<TreeItem> Nodes
		{
			get
			{
				return new ReadOnlyObservableCollection<TreeItem>(this._nodes);
			}
		}

		// Token: 0x17000089 RID: 137
		// (get) Token: 0x06000198 RID: 408 RVA: 0x00007BAE File Offset: 0x00005DAE
		public bool HasChild
		{
			get
			{
				return this._nodes.Count > 0;
			}
		}

		// Token: 0x1700008A RID: 138
		// (get) Token: 0x06000199 RID: 409 RVA: 0x00007BC1 File Offset: 0x00005DC1
		// (set) Token: 0x0600019A RID: 410 RVA: 0x00007BC9 File Offset: 0x00005DC9
		public PackageKey ProgramKey { get; set; }

		// Token: 0x0600019B RID: 411 RVA: 0x00007BD2 File Offset: 0x00005DD2
		public void AddNode(TreeItem element)
		{
			this.AddNode(element, -1);
		}

		// Token: 0x0600019C RID: 412 RVA: 0x00007BDC File Offset: 0x00005DDC
		public void AddNode(TreeItem element, int assignedIndex)
		{
			if (element == null)
			{
				return;
			}
			if (this._nodes == null)
			{
				this._nodes = new TreeItemCollection();
			}
			element.ProgramKey = this.ProgramKey;
			if ((this.Type == DefinitionType.NULL && this.IsFolder) || string.IsNullOrEmpty(element.ReferenceName))
			{
				element.Parent = this;
				if (-1 == assignedIndex)
				{
					element.SortIndex = ((this._nodes.Count > 0) ? (this._nodes[this._nodes.Count - 1].SortIndex + 1) : 1);
					this._nodes.Add(element);
				}
				else
				{
					this._nodes.Insert(assignedIndex, element);
				}
				element.IsExpanded = element.IsFolder;
				for (int i = 0; i < this._nodes.Count; i++)
				{
					TreeItem treeItem = this._nodes[i];
					int num = i + 1;
					if (treeItem.SortIndex != num && !treeItem.IsFolder)
					{
						treeItem.SortIndex = num;
					}
				}
				return;
			}
			if (assignedIndex != -1)
			{
				this._nodes.Insert(assignedIndex, element);
				for (int j = 0; j < this._nodes.Count; j++)
				{
					TreeItem treeItem2 = this._nodes[j];
					int num2 = j + 1;
					if (treeItem2.SortIndex != num2 && !treeItem2.IsFolder)
					{
						treeItem2.SortIndex = num2;
					}
				}
				return;
			}
			switch (element.Type)
			{
			case DefinitionType.FUNCTION:
			case DefinitionType.DIALOG:
			case DefinitionType.REPORT:
			{
				TreeItem child = this.GetChild(element.Type);
				if (child == null)
				{
					return;
				}
				element.Parent = child;
				child.AddNode(element, element.SortIndex);
				return;
			}
			default:
				return;
			}
		}

		// Token: 0x0600019D RID: 413 RVA: 0x00007D98 File Offset: 0x00005F98
		private TreeItem GetChild(DefinitionType type)
		{
			string targetName = type.Description();
			return this._nodes.Where<TreeItem>((TreeItem t) => t.Type == DefinitionType.NULL && t.Name == targetName).ElementAtOrDefault<TreeItem>(0);
		}

		// Token: 0x0600019E RID: 414 RVA: 0x00007DDC File Offset: 0x00005FDC
		public void Clear()
		{
			int i = 0;
			while (i < this._nodes.Count)
			{
				TreeItem treeItem = this._nodes.ElementAtOrDefault<TreeItem>(i);
				if (treeItem.IsFolder || string.IsNullOrEmpty(treeItem.ReferenceName))
				{
					treeItem.Clear();
					i++;
				}
				else
				{
					this._nodes.Remove(treeItem);
				}
			}
		}

		// Token: 0x0600019F RID: 415 RVA: 0x00007E36 File Offset: 0x00006036
		public TreeItem Clone()
		{
			return base.MemberwiseClone() as TreeItem;
		}

		// Token: 0x060001A0 RID: 416 RVA: 0x00007E44 File Offset: 0x00006044
		public void Update(string referenceName, TreeItem newItem)
		{
			TreeItem child = this.GetChild(newItem.Type);
			if (child == null)
			{
				return;
			}
			for (int i = 0; i < child.Nodes.Count; i++)
			{
				TreeItem treeItem = child.Nodes[i];
				if (treeItem.ReferenceName == referenceName)
				{
					newItem.Parent = treeItem.Parent;
					child.Replace(i, newItem);
				}
			}
		}

		// Token: 0x060001A1 RID: 417 RVA: 0x00007EA7 File Offset: 0x000060A7
		private void Replace(int index, TreeItem newChild)
		{
			this._nodes.RemoveAt(index);
			this._nodes.Insert(newChild.SortIndex, newChild);
		}

		// Token: 0x060001A2 RID: 418 RVA: 0x00007EC7 File Offset: 0x000060C7
		public void RemoveAt(int index)
		{
			this.RemoveThenResort(this._nodes[index]);
		}

		// Token: 0x060001A3 RID: 419 RVA: 0x00007EDC File Offset: 0x000060DC
		private void RemoveThenResort(TreeItem itemToDelete)
		{
			this._nodes.Remove(itemToDelete);
			for (int i = 0; i < this._nodes.Count; i++)
			{
				TreeItem treeItem = this._nodes[i];
				int num = i + 1;
				if (treeItem.SortIndex != num && !treeItem.IsFolder)
				{
					treeItem.SortIndex = num;
				}
			}
		}

		// Token: 0x060001A4 RID: 420 RVA: 0x00007F38 File Offset: 0x00006138
		public void Remove(TreeItem itemToDelete)
		{
			if (itemToDelete == null)
			{
				return;
			}
			if (this._nodes.Contains(itemToDelete))
			{
				this.RemoveThenResort(itemToDelete);
				return;
			}
			TreeItem child = this.GetChild(itemToDelete.Type);
			child.Remove(itemToDelete);
		}

		// Token: 0x1700008B RID: 139
		// (get) Token: 0x060001A5 RID: 421 RVA: 0x00007F73 File Offset: 0x00006173
		// (set) Token: 0x060001A6 RID: 422 RVA: 0x00007F7B File Offset: 0x0000617B
		public bool IsSelected
		{
			get
			{
				return this._isSelected;
			}
			set
			{
				this._isSelected = value;
				this.RaisePropertyChanged("IsSelected");
			}
		}

		// Token: 0x060001A7 RID: 423 RVA: 0x00007F90 File Offset: 0x00006190
		public void PublishSelectedEvent()
		{
			this.IsSelected = true;
			if (!this.IsFolder)
			{
				FunctionSelectedModel modelSequence = this.GetModelSequence(this, null);
				EventAggregatorManager.Get(this.ProgramKey).GetEvent<FunctionSelectedEvent>().Publish(modelSequence);
			}
		}

		// Token: 0x060001A8 RID: 424 RVA: 0x00007FCC File Offset: 0x000061CC
		private FunctionSelectedModel GetModelSequence(TreeItem node, FunctionSelectedModel child)
		{
			if (node == null || string.IsNullOrEmpty(node.Name))
			{
				return null;
			}
			FunctionSelectedModel functionSelectedModel = new FunctionSelectedModel
			{
				ProgramKey = (Application.Current.MainWindow.Tag as PackageKey),
				Name = node.Name,
				Child = child,
				Target = node.SearchName,
				Type = node.Type
			};
			if (functionSelectedModel.Type == DefinitionType.NULL)
			{
				functionSelectedModel.Target = string.Format("{0}[^\\S]", functionSelectedModel.Target);
			}
			if (node.Parent != null && node.Scope == Scope.NULL)
			{
				return this.GetModelSequence(node.Parent, functionSelectedModel);
			}
			return functionSelectedModel;
		}

		// Token: 0x1700008C RID: 140
		// (get) Token: 0x060001A9 RID: 425 RVA: 0x00008075 File Offset: 0x00006275
		// (set) Token: 0x060001AA RID: 426 RVA: 0x0000807D File Offset: 0x0000627D
		public bool IsExpanded
		{
			get
			{
				return this._isExpanded;
			}
			set
			{
				if (this.Parent == null)
				{
					return;
				}
				this._isExpanded = value;
				if (this._isExpanded && this.Parent != null)
				{
					this.Parent.IsExpanded = true;
				}
				this.RaisePropertyChanged("IsExpanded");
			}
		}

		// Token: 0x1700008D RID: 141
		// (get) Token: 0x060001AB RID: 427 RVA: 0x000080B6 File Offset: 0x000062B6
		// (set) Token: 0x060001AC RID: 428 RVA: 0x000080BE File Offset: 0x000062BE
		public int Level
		{
			get
			{
				return this._level;
			}
			private set
			{
				this._level = value;
			}
		}

		// Token: 0x1400000B RID: 11
		// (add) Token: 0x060001AD RID: 429 RVA: 0x000080C8 File Offset: 0x000062C8
		// (remove) Token: 0x060001AE RID: 430 RVA: 0x00008100 File Offset: 0x00006300
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060001AF RID: 431 RVA: 0x00008135 File Offset: 0x00006335
		private void RaisePropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x1700008E RID: 142
		// (get) Token: 0x060001B0 RID: 432 RVA: 0x00008159 File Offset: 0x00006359
		public ICommand DeleteCommand
		{
			get
			{
				if (this._deleteCommand == null)
				{
					this._deleteCommand = new RelayCommand(delegate(object p)
					{
						this.OnDelete();
					});
				}
				return this._deleteCommand;
			}
		}

		// Token: 0x060001B1 RID: 433 RVA: 0x00008180 File Offset: 0x00006380
		private void OnDelete()
		{
			EventController.GetInstance().GetEvent<DeleteFunctionEvent>().Publish(this.ReferenceName);
			EventAggregatorManager.Get(this.ProgramKey).GetEvent<CodeChangedEvent>().Publish(this.ProgramKey);
		}

		// Token: 0x1700008F RID: 143
		// (get) Token: 0x060001B2 RID: 434 RVA: 0x000081DC File Offset: 0x000063DC
		public bool CanRemove
		{
			get
			{
				if (!ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).VerifyAdjustFunctionSort)
				{
					return false;
				}
				if (this.IsFolder)
				{
					return false;
				}
				if (string.IsNullOrEmpty(this.ReferenceName))
				{
					return false;
				}
				AddPointModel addPointModel = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).AddPoints.Where<AddPointModel>((AddPointModel m) => m.Name == this.ReferenceName && m.IsNew && (m.Status & Status.DELETE) == Status.NULL).ElementAtOrDefault<AddPointModel>(0);
				return addPointModel != null && addPointModel.IsEditable;
			}
		}

		// Token: 0x060001B3 RID: 435 RVA: 0x00008254 File Offset: 0x00006454
		public void Dispose()
		{
			if (this.HasChild)
			{
				for (int i = 0; i < this._nodes.Count; i++)
				{
					TreeItem treeItem = this._nodes[i];
					treeItem.Dispose();
				}
			}
		}

		// Token: 0x17000090 RID: 144
		// (get) Token: 0x060001B4 RID: 436 RVA: 0x00008292 File Offset: 0x00006492
		public Type DragType
		{
			get
			{
				return typeof(TreeItem);
			}
		}

		// Token: 0x17000091 RID: 145
		// (get) Token: 0x060001B5 RID: 437 RVA: 0x0000829E File Offset: 0x0000649E
		public bool CanDrag
		{
			get
			{
				return this.CanRemove && !string.IsNullOrEmpty(this.ReferenceName);
			}
		}

		// Token: 0x17000092 RID: 146
		// (get) Token: 0x060001B6 RID: 438 RVA: 0x000082BA File Offset: 0x000064BA
		public Type AllowType
		{
			get
			{
				return typeof(TreeItem);
			}
		}

		// Token: 0x060001B7 RID: 439 RVA: 0x000082C6 File Offset: 0x000064C6
		public void DropOver(DragEventArgs dragEvetnArgs)
		{
		}

		// Token: 0x060001B8 RID: 440 RVA: 0x000082DC File Offset: 0x000064DC
		private int FindItemIndex(TreeItem item)
		{
			TreeItem treeItem = this._nodes.Where<TreeItem>((TreeItem i) => i == item).ElementAtOrDefault<TreeItem>(0);
			if (treeItem == null)
			{
				return -1;
			}
			return treeItem.SortIndex;
		}

		// Token: 0x060001B9 RID: 441 RVA: 0x00008320 File Offset: 0x00006520
		public TreeItem Find(string name)
		{
			if (!this.HasChild)
			{
				return null;
			}
			foreach (TreeItem treeItem in this._nodes)
			{
				if (treeItem.Name == name)
				{
					return treeItem;
				}
			}
			return null;
		}

		// Token: 0x060001BA RID: 442 RVA: 0x00008388 File Offset: 0x00006588
		public void Drop(IDragable drag)
		{
			TreeItem treeItem = drag as TreeItem;
			try
			{
				this.Parent.Remove(treeItem);
				TreeItem treeItem2 = treeItem.Clone();
				int num = this.Parent.FindItemIndex(this);
				this.Parent.AddNode(treeItem2, num + 1);
			}
			catch (Exception ex)
			{
				throw new Exception(string.Format(Application.Current.FindResource("Message_NodeMoveError") as string, ex.Message));
			}
		}

		// Token: 0x17000093 RID: 147
		// (get) Token: 0x060001BB RID: 443 RVA: 0x00008404 File Offset: 0x00006604
		public bool CanDrop
		{
			get
			{
				return this.CanRemove;
			}
		}

		// Token: 0x060001BC RID: 444 RVA: 0x0000840C File Offset: 0x0000660C
		public bool CheckDropable(IDragable dragable)
		{
			TreeItem treeItem = dragable as TreeItem;
			return treeItem != null && treeItem.Type == this.Type && this.CanRemove;
		}

		// Token: 0x040000A2 RID: 162
		private int _sorIndex = 1;

		// Token: 0x040000A3 RID: 163
		private string _name;

		// Token: 0x040000A4 RID: 164
		private string _displayName = string.Empty;

		// Token: 0x040000A5 RID: 165
		private TreeItem _parent;

		// Token: 0x040000A6 RID: 166
		private string _description;

		// Token: 0x040000A7 RID: 167
		private Scope _scope = Scope.PUBLIC;

		// Token: 0x040000A8 RID: 168
		private bool _isVisible = true;

		// Token: 0x040000A9 RID: 169
		private TreeItemCollection _nodes = new TreeItemCollection();

		// Token: 0x040000AA RID: 170
		private bool _isSelected;

		// Token: 0x040000AB RID: 171
		private bool _isExpanded = true;

		// Token: 0x040000AC RID: 172
		private int _level = 1;

		// Token: 0x040000AE RID: 174
		private RelayCommand _deleteCommand;
	}
}
