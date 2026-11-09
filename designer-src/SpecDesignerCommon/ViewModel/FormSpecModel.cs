using System;
using System.ComponentModel;
using System.Xml.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200006D RID: 109
	public class FormSpecModel : INotifyPropertyChanged, ISpecSearchable
	{
		// Token: 0x17000110 RID: 272
		// (get) Token: 0x06000412 RID: 1042 RVA: 0x00012C40 File Offset: 0x00010E40
		// (set) Token: 0x06000413 RID: 1043 RVA: 0x00012C48 File Offset: 0x00010E48
		public PackageKey Key { get; private set; }

		// Token: 0x17000111 RID: 273
		// (get) Token: 0x06000414 RID: 1044 RVA: 0x00012C51 File Offset: 0x00010E51
		// (set) Token: 0x06000415 RID: 1045 RVA: 0x00012C5C File Offset: 0x00010E5C
		public string Name
		{
			get
			{
				return this._name;
			}
			set
			{
				if (this._name != value && !this.IsCited)
				{
					if (this.SpecField != null)
					{
						this.SpecField.SetName(this, value);
					}
					if (this.SpecAction != null)
					{
						this.SpecAction.SetName(this, value);
					}
					if (this.SpecHelpCode != null)
					{
						this.SpecHelpCode.SetName(this, value);
					}
					if (this.SpecMultiLang != null)
					{
						this.SpecMultiLang.SetName(this, value);
					}
					if (this.SpecProgRel != null)
					{
						this.SpecProgRel.SetName(this, value);
					}
					if (this.SpecTree != null)
					{
						this.SpecTree.SetName(this, value);
					}
					if (this.SpecReference != null)
					{
						this.SpecReference.SetName(this, value);
					}
					if (this.SpecExcludeNode != null)
					{
						this.SpecExcludeNode.SetName(this, value);
					}
					this.GeneroComponent.SetAttribute("name", value);
					this._name = value;
				}
				this.OnPropertyChanged("Name");
				this.GeneroComponent.OnPropertyChanged("LocalString");
			}
		}

		// Token: 0x17000112 RID: 274
		// (get) Token: 0x06000416 RID: 1046 RVA: 0x00012D60 File Offset: 0x00010F60
		// (set) Token: 0x06000417 RID: 1047 RVA: 0x00012D68 File Offset: 0x00010F68
		public XmlElement GeneroComponent { get; private set; }

		// Token: 0x17000113 RID: 275
		// (get) Token: 0x06000418 RID: 1048 RVA: 0x00012D71 File Offset: 0x00010F71
		public ComponentType Type
		{
			get
			{
				if (this.GeneroComponent != null)
				{
					return this.GeneroComponent.Type;
				}
				return ComponentType.Unknown;
			}
		}

		// Token: 0x17000114 RID: 276
		// (get) Token: 0x06000419 RID: 1049 RVA: 0x00012D88 File Offset: 0x00010F88
		public SpecNodeType SpecNodeType
		{
			get
			{
				return FormSpecModel.GetNodeType(this.GeneroComponent);
			}
		}

		// Token: 0x17000115 RID: 277
		// (get) Token: 0x0600041A RID: 1050 RVA: 0x00012D98 File Offset: 0x00010F98
		// (set) Token: 0x0600041B RID: 1051 RVA: 0x00012E34 File Offset: 0x00011034
		public bool IsCited
		{
			get
			{
				if (this.SpecField != null)
				{
					return this.SpecField.IsCited;
				}
				if (this.SpecAction != null)
				{
					return this.SpecAction.IsCited;
				}
				if (this.SpecHelpCode != null)
				{
					return this.SpecHelpCode.IsCited;
				}
				if (this.SpecMultiLang != null)
				{
					return this.SpecMultiLang.IsCited;
				}
				if (this.SpecProgRel != null)
				{
					return this.SpecProgRel.IsCited;
				}
				if (this.SpecReference != null)
				{
					return this.SpecReference.IsCited;
				}
				return this.SpecTree != null && this.SpecTree.IsCited;
			}
			set
			{
				if (this.SpecField != null && value != this.SpecField.IsCited)
				{
					this.SpecField.IsCited = value;
				}
				if (this.SpecAction != null && value != this.SpecAction.IsCited)
				{
					this.SpecAction.IsCited = value;
				}
				if (this.SpecHelpCode != null && value != this.SpecHelpCode.IsCited)
				{
					this.SpecHelpCode.IsCited = value;
				}
				if (this.SpecMultiLang != null && value != this.SpecMultiLang.IsCited)
				{
					this.SpecMultiLang.IsCited = value;
				}
				if (this.SpecProgRel != null && value != this.SpecProgRel.IsCited)
				{
					this.SpecProgRel.IsCited = value;
				}
				if (this.SpecReference != null && value != this.SpecReference.IsCited)
				{
					this.SpecReference.IsCited = value;
				}
				if (this.SpecTree != null && value != this.SpecTree.IsCited)
				{
					this.SpecTree.IsCited = value;
				}
				this.OnPropertyChanged("IsCited");
			}
		}

		// Token: 0x17000116 RID: 278
		// (get) Token: 0x0600041C RID: 1052 RVA: 0x00012F3C File Offset: 0x0001113C
		public XElement CitedSpec
		{
			get
			{
				if (this.SpecField != null)
				{
					return this.SpecField.CitedSpec;
				}
				if (this.SpecAction != null)
				{
					return this.SpecAction.CitedSpec;
				}
				if (this.SpecHelpCode != null)
				{
					return this.SpecHelpCode.CitedSpec;
				}
				if (this.SpecMultiLang != null)
				{
					return this.SpecMultiLang.CitedSpec;
				}
				if (this.SpecProgRel != null)
				{
					return this.SpecProgRel.CitedSpec;
				}
				if (this.SpecReference != null)
				{
					return this.SpecReference.CitedSpec;
				}
				if (this.SpecTree != null)
				{
					return this.SpecTree.CitedSpec;
				}
				return null;
			}
		}

		// Token: 0x17000117 RID: 279
		// (get) Token: 0x0600041D RID: 1053 RVA: 0x00012FD6 File Offset: 0x000111D6
		public SpecExcludeNode SpecExcludeNode
		{
			get
			{
				return this._excluedeNode;
			}
		}

		// Token: 0x17000118 RID: 280
		// (get) Token: 0x0600041E RID: 1054 RVA: 0x00012FDE File Offset: 0x000111DE
		// (set) Token: 0x0600041F RID: 1055 RVA: 0x00012FE8 File Offset: 0x000111E8
		public bool IsExcluded
		{
			get
			{
				return this._isExcluded;
			}
			set
			{
				if (value == this._isExcluded)
				{
					return;
				}
				ExcludedUndoRedoCommand excludedUndoRedoCommand = new ExcludedUndoRedoCommand(this, value);
				SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(excludedUndoRedoCommand);
			}
		}

		// Token: 0x06000420 RID: 1056 RVA: 0x0001301D File Offset: 0x0001121D
		internal void SetExcluded(bool isExcluded)
		{
			if (this._isExcluded == isExcluded)
			{
				return;
			}
			this.SetExcluded(SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo, isExcluded);
		}

		// Token: 0x06000421 RID: 1057 RVA: 0x00013048 File Offset: 0x00011248
		internal void SetExcluded(SpecificationInfo info, bool isExcluded)
		{
			this._isExcluded = isExcluded;
			if (this._excluedeNode == null)
			{
				if (this._isExcluded)
				{
					this._excluedeNode = SpecExcludeNode.Create(info, this.Name);
					this._excluedeNode.Status = SpecStatus.MODIFY;
					return;
				}
			}
			else
			{
				this._excluedeNode.Status = (this._isExcluded ? SpecStatus.MODIFY : SpecStatus.DELETE);
			}
		}

		// Token: 0x17000119 RID: 281
		// (get) Token: 0x06000422 RID: 1058 RVA: 0x000130A2 File Offset: 0x000112A2
		// (set) Token: 0x06000423 RID: 1059 RVA: 0x000130AA File Offset: 0x000112AA
		public SpecFieldNode SpecField { get; set; }

		// Token: 0x1700011A RID: 282
		// (get) Token: 0x06000424 RID: 1060 RVA: 0x000130B3 File Offset: 0x000112B3
		// (set) Token: 0x06000425 RID: 1061 RVA: 0x000130BB File Offset: 0x000112BB
		public SpecActionNode SpecAction { get; set; }

		// Token: 0x1700011B RID: 283
		// (get) Token: 0x06000426 RID: 1062 RVA: 0x000130C4 File Offset: 0x000112C4
		// (set) Token: 0x06000427 RID: 1063 RVA: 0x000130CC File Offset: 0x000112CC
		public SpecHelpCodeNode SpecHelpCode { get; set; }

		// Token: 0x1700011C RID: 284
		// (get) Token: 0x06000428 RID: 1064 RVA: 0x000130D5 File Offset: 0x000112D5
		// (set) Token: 0x06000429 RID: 1065 RVA: 0x000130DD File Offset: 0x000112DD
		public SpecMultiLangNode SpecMultiLang { get; set; }

		// Token: 0x1700011D RID: 285
		// (get) Token: 0x0600042A RID: 1066 RVA: 0x000130E6 File Offset: 0x000112E6
		// (set) Token: 0x0600042B RID: 1067 RVA: 0x000130EE File Offset: 0x000112EE
		public SpecProgRelNode SpecProgRel { get; set; }

		// Token: 0x1700011E RID: 286
		// (get) Token: 0x0600042C RID: 1068 RVA: 0x000130F7 File Offset: 0x000112F7
		// (set) Token: 0x0600042D RID: 1069 RVA: 0x000130FF File Offset: 0x000112FF
		public SpecReferenceNode SpecReference { get; set; }

		// Token: 0x1700011F RID: 287
		// (get) Token: 0x0600042E RID: 1070 RVA: 0x00013108 File Offset: 0x00011308
		// (set) Token: 0x0600042F RID: 1071 RVA: 0x00013110 File Offset: 0x00011310
		public SpecTreeNode SpecTree { get; set; }

		// Token: 0x06000430 RID: 1072 RVA: 0x00013119 File Offset: 0x00011319
		internal void SetExcludedNode(XElement element)
		{
			if (this._excluedeNode != null)
			{
				throw new Exception("SpecExcludeNode already exist");
			}
			this._excluedeNode = new SpecExcludeNode(this.Key, element);
			this._isExcluded = true;
		}

		// Token: 0x06000431 RID: 1073 RVA: 0x00013148 File Offset: 0x00011348
		internal void SetSpecNode(AbstractSpecNode node)
		{
			if (node is SpecFieldNode)
			{
				this.SetFieldNode(node as SpecFieldNode);
				return;
			}
			if (node is SpecActionNode)
			{
				this.SetActionNode(node as SpecActionNode);
				return;
			}
			if (node is SpecHelpCodeNode)
			{
				this.SetHelpCodeNode(node as SpecHelpCodeNode);
				return;
			}
			if (node is SpecMultiLangNode)
			{
				this.SetMultiLangNode(node as SpecMultiLangNode);
				return;
			}
			if (node is SpecProgRelNode)
			{
				this.SetProgRelNode(node as SpecProgRelNode);
				return;
			}
			if (node is SpecReferenceNode)
			{
				this.SetReferenceNode(node as SpecReferenceNode);
				return;
			}
			if (node is SpecTreeNode)
			{
				this.SetTreeNode(node as SpecTreeNode);
			}
		}

		// Token: 0x06000432 RID: 1074 RVA: 0x000131E7 File Offset: 0x000113E7
		private void SetFieldNode(SpecFieldNode node)
		{
			if (this.SpecField != null)
			{
				throw new Exception("SpecFieldNode already exist");
			}
			node.Widget = this.GeneroComponent.Type.ToString();
			this.SpecField = node;
		}

		// Token: 0x06000433 RID: 1075 RVA: 0x0001321E File Offset: 0x0001141E
		private void SetActionNode(SpecActionNode node)
		{
			if (this.SpecAction != null)
			{
				throw new Exception("SpecActionNode already exist");
			}
			this.SpecAction = node;
		}

		// Token: 0x06000434 RID: 1076 RVA: 0x0001323A File Offset: 0x0001143A
		private void SetHelpCodeNode(SpecHelpCodeNode node)
		{
			if (this.SpecHelpCode != null)
			{
				throw new Exception("SpecHelpCodeNode already exist");
			}
			this.SpecHelpCode = node;
		}

		// Token: 0x06000435 RID: 1077 RVA: 0x00013256 File Offset: 0x00011456
		private void SetMultiLangNode(SpecMultiLangNode node)
		{
			if (this.SpecMultiLang != null)
			{
				throw new Exception("SpecMultiLangNode already exist");
			}
			this.SpecMultiLang = node;
		}

		// Token: 0x06000436 RID: 1078 RVA: 0x00013272 File Offset: 0x00011472
		private void SetProgRelNode(SpecProgRelNode node)
		{
			if (this.SpecProgRel != null)
			{
				throw new Exception("SpecProgRelNode already exist");
			}
			this.SpecProgRel = node;
		}

		// Token: 0x06000437 RID: 1079 RVA: 0x0001328E File Offset: 0x0001148E
		private void SetReferenceNode(SpecReferenceNode node)
		{
			if (this.SpecReference != null)
			{
				throw new Exception("SpecReferenceNode already exist");
			}
			this.SpecReference = node;
		}

		// Token: 0x06000438 RID: 1080 RVA: 0x000132AC File Offset: 0x000114AC
		private void SetTreeNode(SpecTreeNode node)
		{
			if (this.SpecTree != null)
			{
				throw new Exception("SpecTreeNode already exist");
			}
			if (!this.Name.Equals("s_browse", StringComparison.CurrentCultureIgnoreCase))
			{
				node.Widget = this.GeneroComponent.Type.ToString();
			}
			this.SpecTree = node;
		}

		// Token: 0x06000439 RID: 1081 RVA: 0x00013301 File Offset: 0x00011501
		public FormSpecModel(PackageKey programKey, XmlElement formModel)
		{
			this.GeneroComponent = formModel;
			this.Key = programKey;
			this._name = formModel.Name;
		}

		// Token: 0x0600043A RID: 1082 RVA: 0x00013330 File Offset: 0x00011530
		public static SpecNodeType GetNodeType(XmlElement ce)
		{
			if ("s_browse" == ce.Name)
			{
				return SpecNodeType.TREE;
			}
			if (ce.IsCantDel)
			{
				return SpecNodeType.NONE;
			}
			SpecNodeType specNodeType = ((ce.GetAttribute("colName") == null) ? SpecNodeType.FORMONLY : SpecNodeType.FIELD);
			ComponentType type = ce.Type;
			if (type <= ComponentType.Tree)
			{
				if (type != ComponentType.Form)
				{
					if (type == ComponentType.Tree)
					{
						if ("s_browse" == ce.Name)
						{
							specNodeType = SpecNodeType.TREE;
						}
					}
				}
				else
				{
					specNodeType = SpecNodeType.NONE;
				}
			}
			else if (type != ComponentType.Edit)
			{
				switch (type)
				{
				case ComponentType.Button:
					specNodeType = (("button_qrystr" == ce.GetAttribute("style")) ? SpecNodeType.PROGREL : SpecNodeType.ACTION);
					break;
				case ComponentType.ButtonEdit:
					if ("16/langmodify.png".Equals(ce.GetAttribute("image")))
					{
						specNodeType = SpecNodeType.MULTILANG;
					}
					break;
				default:
					if (type == ComponentType.FFLabel)
					{
						if ("sync".Equals(ce.GetAttribute("tag")) && ce.Parent != null && (ce.Parent.Type == ComponentType.Table || ce.Parent.Type == ComponentType.Tree))
						{
							specNodeType = SpecNodeType.PROGREL;
						}
						else if ("reference".Equals(ce.GetAttribute("style")))
						{
							specNodeType = SpecNodeType.REFERENCE;
						}
					}
					break;
				}
			}
			else if ("reference".Equals(ce.GetAttribute("style")) && ce.Parent != null && (ce.Parent.Type == ComponentType.Table || ce.Parent.Type == ComponentType.Tree))
			{
				specNodeType = SpecNodeType.REFERENCE;
			}
			return specNodeType;
		}

		// Token: 0x17000120 RID: 288
		// (get) Token: 0x0600043B RID: 1083 RVA: 0x000134B0 File Offset: 0x000116B0
		public AbstractSpecNode SpecNode
		{
			get
			{
				switch (this.SpecNodeType)
				{
				case SpecNodeType.FIELD:
					return this.SpecField;
				case SpecNodeType.ACTION:
					return this.SpecAction;
				case SpecNodeType.PROGREL:
					return this.SpecProgRel;
				case SpecNodeType.REFERENCE:
					return this.SpecReference;
				case SpecNodeType.MULTILANG:
					return this.SpecMultiLang;
				}
				return null;
			}
		}

		// Token: 0x17000121 RID: 289
		// (get) Token: 0x0600043C RID: 1084 RVA: 0x00013510 File Offset: 0x00011710
		// (set) Token: 0x0600043D RID: 1085 RVA: 0x00013564 File Offset: 0x00011764
		public SpecStatus SpecNodeStatus
		{
			get
			{
				if (this.SpecNode == null)
				{
					return SpecStatus.NULL;
				}
				SpecStatus status = this.SpecNode.Status;
				if (status == SpecStatus.MODIFY)
				{
					return SpecStatus.MODIFY;
				}
				if (!string.Equals("Y", this.SpecNode.GetAttribute("ch"), StringComparison.CurrentCultureIgnoreCase))
				{
					return this.SpecNode.Status;
				}
				return SpecStatus.MODIFY;
			}
			set
			{
				if (this.SpecNode == null)
				{
					return;
				}
				if (value != SpecStatus.DELETE && value != SpecStatus.MODIFY)
				{
					return;
				}
				if (this.SpecField != null && this.SpecField.Status != SpecStatus.CREATE)
				{
					this.SpecField.Status = value;
				}
				if (this.SpecAction != null && this.SpecAction.Status != SpecStatus.CREATE)
				{
					this.SpecAction.Status = value;
				}
				if (this.SpecHelpCode != null && this.SpecHelpCode.Status != SpecStatus.CREATE)
				{
					this.SpecHelpCode.Status = value;
				}
				if (this.SpecMultiLang != null && this.SpecMultiLang.Status != SpecStatus.CREATE)
				{
					this.SpecMultiLang.Status = value;
				}
				if (this.SpecProgRel != null && this.SpecProgRel.Status != SpecStatus.CREATE)
				{
					this.SpecProgRel.Status = value;
				}
				if (this.SpecTree != null && this.SpecTree.Status != SpecStatus.CREATE)
				{
					this.SpecTree.Status = value;
				}
				if (this.SpecReference != null && this.SpecReference.Status != SpecStatus.CREATE)
				{
					this.SpecReference.Status = value;
				}
				if (this.SpecExcludeNode != null)
				{
					this.SpecExcludeNode.Status = value;
				}
			}
		}

		// Token: 0x0600043E RID: 1086 RVA: 0x00013687 File Offset: 0x00011887
		public override string ToString()
		{
			return string.Format("{0}, {1}", this.Name, this.SpecNodeType.ToString());
		}

		// Token: 0x14000011 RID: 17
		// (add) Token: 0x0600043F RID: 1087 RVA: 0x000136AC File Offset: 0x000118AC
		// (remove) Token: 0x06000440 RID: 1088 RVA: 0x000136E4 File Offset: 0x000118E4
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000441 RID: 1089 RVA: 0x00013719 File Offset: 0x00011919
		internal void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
			EventAggregatorManager.Get(this.Key).GetEvent<FormPropertyChangedEvent>().Publish(this.Key);
		}

		// Token: 0x06000442 RID: 1090 RVA: 0x00013750 File Offset: 0x00011950
		public bool Contains(string tag, string value)
		{
			throw new NotImplementedException();
		}

		// Token: 0x06000443 RID: 1091 RVA: 0x00013757 File Offset: 0x00011957
		public void ChangeAttributeValue(string tag, string oldValue, string newValue)
		{
			throw new NotImplementedException();
		}

		// Token: 0x04000192 RID: 402
		private string _name = string.Empty;

		// Token: 0x04000193 RID: 403
		private SpecExcludeNode _excluedeNode;

		// Token: 0x04000194 RID: 404
		private bool _isExcluded;
	}
}
